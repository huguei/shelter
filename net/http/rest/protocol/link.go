// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package protocol describes the REST protocol
package protocol

// List of link type obtained from IANA at
// http://www.iana.org/assignments/link-relations/link-relations.xml
// For specific details from each link type visit the IANA website and check the RFC
// related to the link type
const (
	LinkTypeAbout              LinkType = "about"
	LinkTypeAlternate          LinkType = "alternate"
	LinkTypeAppendix           LinkType = "appendix"
	LinkTypeArchives           LinkType = "archives"
	LinkTypeAuthor             LinkType = "author"
	LinkTypeBookmark           LinkType = "bookmark"
	LinkTypeCanonical          LinkType = "canonical"
	LinkTypeChapter            LinkType = "chapter"
	LinkTypeCollection         LinkType = "collection"
	LinkTypeContents           LinkType = "contents"
	LinkTypeCopyright          LinkType = "copyright"
	LinkTypeCreateForm         LinkType = "create-form"
	LinkTypeCurrent            LinkType = "current"
	LinkTypeDescribedby        LinkType = "describedby"
	LinkTypeDescribes          LinkType = "describes"
	LinkTypeDisclosure         LinkType = "disclosure"
	LinkTypeDuplicate          LinkType = "duplicate"
	LinkTypeEdit               LinkType = "edit"
	LinkTypeEditForm           LinkType = "edit-form"
	LinkTypeEditMedia          LinkType = "edit-media"
	LinkTypeEnclosure          LinkType = "enclosure"
	LinkTypeFirst              LinkType = "first"
	LinkTypeGlossary           LinkType = "glossary"
	LinkTypeHelp               LinkType = "help"
	LinkTypeHosts              LinkType = "hosts"
	LinkTypeHub                LinkType = "hub"
	LinkTypeIcon               LinkType = "icon"
	LinkTypeIndex              LinkType = "index"
	LinkTypeItem               LinkType = "item"
	LinkTypeLast               LinkType = "last"
	LinkTypeLatestVersion      LinkType = "latest-version"
	LinkTypeLicense            LinkType = "license"
	LinkTypeLrdd               LinkType = "lrdd"
	LinkTypeMemento            LinkType = "memento"
	LinkTypeMonitor            LinkType = "monitor"
	LinkTypeMonitorGroup       LinkType = "monitor-group"
	LinkTypeNext               LinkType = "next"
	LinkTypeNextArchive        LinkType = "next-archive"
	LinkTypeNofollow           LinkType = "nofollow"
	LinkTypeNoreferrer         LinkType = "noreferrer"
	LinkTypeOriginal           LinkType = "original"
	LinkTypePayment            LinkType = "payment"
	LinkTypePredecessorVersion LinkType = "predecessor-version"
	LinkTypePrefetch           LinkType = "prefetch"
	LinkTypePrev               LinkType = "prev"
	LinkTypePreview            LinkType = "preview"
	LinkTypePrevious           LinkType = "previous"
	LinkTypePrevArchive        LinkType = "prev-archive"
	LinkTypePrivacyPolicy      LinkType = "privacy-policy"
	LinkTypeProfile            LinkType = "profile"
	LinkTypeRelated            LinkType = "related"
	LinkTypeReplies            LinkType = "replies"
	LinkTypeSearch             LinkType = "search"
	LinkTypeSection            LinkType = "section"
	LinkTypeSelf               LinkType = "self"
	LinkTypeService            LinkType = "service"
	LinkTypeStart              LinkType = "start"
	LinkTypeStylesheet         LinkType = "stylesheet"
	LinkTypeSubsection         LinkType = "subsection"
	LinkTypeSuccessorVersion   LinkType = "successor-version"
	LinkTypeTag                LinkType = "tag"
	LinkTypeTermsOfService     LinkType = "terms-of-service"
	LinkTypeTimegate           LinkType = "timegate"
	LinkTypeTimemap            LinkType = "timemap"
	LinkTypeType               LinkType = "type"
	LinkTypeUp                 LinkType = "up"
	LinkTypeVersionHistory     LinkType = "version-history"
	LinkTypeVia                LinkType = "via"
	LinkTypeWorkingCopy        LinkType = "working-copy"
	LinkTypeWorkingCopyOf      LinkType = "working-copy-of"
)

// LinkType is a string that represents the type of the link used in the protocol
type LinkType string

// Link structure was created to relate resources in the system. A link can have multiple
// types according to W3C in http://www.w3.org/TR/html401/struct/links.html
type Link struct {
	Types []LinkType `json:"types,omitempty"` // List of link types that can be applied for the same URI
	HRef  string     `json:"href,omitempty"`  // URI for the link
}
