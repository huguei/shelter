#!/usr/bin/env python

# Copyright 2014 Rafael Dantas Justo. All rights reserved.
# Use of this source code is governed by a GPL
# license that can be found in the LICENSE file.

import fnmatch
import os
import subprocess
import sys

def initialChecks():
  if "GOPATH" not in os.environ:
    print("Need to set GOPATH")
    sys.exit(1)

def findPath():
  goPath = os.environ["GOPATH"]
  goPathParts = goPath.split(":")
  for goPathPart in goPathParts:
    projectPath = os.path.join(goPathPart, "src", "github.com",
      "rafaeljusto", "shelter")
    if os.path.exists(projectPath):
      return projectPath

  return ""

def changePath():
  projectPath = findPath()
  if len(projectPath) == 0:
    print("Project not found")
    sys.exit(1)

  os.chdir(projectPath)

def runCoverReport():
  print("\n[[ UNIT TESTS ]]\n")

  goPackages = []
  for root, dirnames, filenames in os.walk("."):
    for filename in fnmatch.filter(filenames, "*_test.go"):
      # TODO: We should test this in Windows
      goPackage = "github.com/rafaeljusto/shelter" + root[1:]
      goPackages.append(goPackage)

  goPackages = set(goPackages)
  goPackages = list(goPackages)
  goPackages.sort()

  success = True

  for goPackage in goPackages:
    packageName = goPackage.replace("/", "-")

    try:
      subprocess.check_call(["go", "install", goPackage])
      output = subprocess.check_output(["go", "test", "-covermode=count", "-coverprofile=" + packageName + ".cover.out", "-cover", goPackage])
      sys.stdout.write(output)

    except subprocess.CalledProcessError:
      success = False

  # http://lk4d4.darth.io/posts/multicover/
  mergeCommand = "|".join([
    "echo 'mode: count' > cover-profile.out && cat *.cover.out",
    "grep -v mode:",
    "sort -r",
    "awk '{if($1 != last) {print $0;last=$1}}' >> cover-profile.out"
  ])

  try:
    subprocess.check_call(["sh", "-c", mergeCommand])
    subprocess.check_call(["go", "tool", "cover", "-html=cover-profile.out"])

  except subprocess.CalledProcessError:
    success = False

  # Remove the temporary file created for the
  # covering reports
  try:
    os.remove("cover-profile.out")

    for goPackage in goPackages:
      filename = goPackage.replace("/", "-") + ".cover.out"

      if os.path.isfile(filename):
        os.remove(filename)

  except OSError:
    pass

  if not success:
    print("Errors during the unit test execution")
    sys.exit(1)

###################################################################

if __name__ == "__main__":
  try:
    initialChecks()
    changePath()
    runCoverReport()
  except KeyboardInterrupt:
    sys.exit(1)
