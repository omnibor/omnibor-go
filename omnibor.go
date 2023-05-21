/*
Package omnibor implements OmniBOR.

Read the spec at https://github.com/omnibor/spec/blob/main/SPEC.md

It is an application of the git DAG, a widely used merkle tree with a flat-file storage format, to the challenge of creating build artifact trees in today’s language-heterogeneous open source environments.
by generating artifact trees at build time, embedding the hash of the tree in produced artifacts, and referencing that hash in the next build step, OmniBOR will enable the zero-end-user-effort creation of verifiable build trees. Furthermore, it will enable launch-time comparison of vulnerability data against a complete artifact tree for both open source and proprietary projects (if vuln data is traceable back to source file).

Objective
It is desirable to enable efficient launch-time comparison of the verifiable and complete build tree of any executable component [1] against a then-current list of undesirable source files [2] which are known to be undesirable, where such a build tree contains unique referents for all sources from which the given executable object was composed.

[1]: binary, dynamically-linked library, container image, etc.

[2]: because vulnerabilities may be discovered between the time an executable is created and the time when it is run, these processes must be decoupled
*/
package omnibor

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/edwarnicke/gitoid"
)

// ArtifactTree provides a common interface that assists with the creation and management of an OmniBOR document.
type ArtifactTree interface {
	Identifier

	// AddReference adds a SHA1+SHA256 based git reference to the current OmniBOR document.
	// obj []byte is the byte array to be tagged in the GitRef.
	// bom Identifier is the omnibor identifier of the artifact tree used to create the object.
	// The resulting reference is based on the GitRef format.
	// It returns an error if the SHA1 or SHA256 implementations fails.
	AddReference(obj []byte, bom Identifier) error

	// AddReferenceFromReader adds a SHA1+SHA256 based git reference to the current OmniBOR document.
	// The resulting reference is based on the GitRef format.
	// The io.Reader will be continuously be read until the reader returns a non-null error.
	// If the io.Reader returns io.EOF, the read is considered to be complete.
	// Any other return value from Reader is an error.
	// The object length must be included.
	// If the amount of bytes read does not match the stated object length, an error is returned.
	AddReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error

	// AddExistingReference adds an existing pre-computed reference
	// The string must be a valid gitoid identifier.
	AddExistingReference(s string) error

	// References Returns a lsit of references in the order it will be printed.
	References() []Reference

	// String Returns the string representation of the OmniBOR.
	String() string
}

type Reference interface {
	// Identity returns the GitRef identity of the object as a hex string.
	Identity() string

	// Bom returns an Identifier representing the dependency tree of the object represented by the Identity
	Bom() Identifier

	// String returns a ArtifactTree entry represented by this Reference.
	String() string
}

func referenceSorter(r1, r2 Reference) bool {
	return r1.Identity() < r2.Identity()
}

type by func(p1, p2 Reference) bool

func (b by) sort(refs []Reference) {
	sorter := &referenceSort{
		refs: refs,
		by:   b,
	}
	sort.Sort(sorter)
}

type reference struct {
	hashType string
	identity string
	bom      Identifier
}

type referenceSort struct {
	refs []Reference
	by   by
}

func (grs *referenceSort) Len() int {
	return len(grs.refs)
}

func (grs *referenceSort) Swap(i, j int) {
	grs.refs[i], grs.refs[j] = grs.refs[j], grs.refs[i]
}

func (grs *referenceSort) Less(i, j int) bool {
	return grs.by(grs.refs[i], grs.refs[j])
}

func (ref reference) Identity() string {
	return ref.identity
}

func (ref reference) Bom() Identifier {
	return ref.Bom()
}

func (ref reference) String() string {
	res := fmt.Sprintf("blob %s", ref.identity)
	if ref.bom != nil {
		res = fmt.Sprintf("%s bom %s", res, ref.bom.Identity())
	}

	res = res + "\n"
	return res
}

type Identifier interface {
	Identity() string
}

type omniBor struct {
	lock          sync.Mutex
	gitRefs       []Reference
	gitoidOptions []gitoid.Option
	hashType      string
}

// NewSha1OmniBOR creates a new ArtifactTree object.
// Thread Safety: none, apply your own controls.
//
// Adding duplicate objects with the same Reference identity results in only one Reference entry.
// References are sorted in ascending order based on their UTF-8 values.
//
// Implementation details:
// Adding a Reference is O(n) to discover duplicates.
// Generating a ArtifactTree is O(n*log(n)) as it sorts the existing refs.
func NewSha1OmniBOR() ArtifactTree {
	return &omniBor{
		hashType: "sha1",
	}
}

func NewSha256OmniBOR() ArtifactTree {
	options := []gitoid.Option{gitoid.WithSha256()}
	return &omniBor{
		gitoidOptions: options,
		hashType:      "sha256",
	}
}

func (srv *omniBor) AddReference(obj []byte, bom Identifier) error {
	reader := bytes.NewBuffer(obj)
	return srv.addGitRef(reader, bom, int64(len(obj)))
}

func (srv *omniBor) AddReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error {
	return srv.addGitRef(reader, bom, objLength)
}

func (srv *omniBor) AddExistingReference(input string) error {
	// if srv is using sha1, check that the input is a valid hex sha1 and length
	// if srv is in sha256 mode, set hashLength to the length of a sha256 hash
	hashLength := 40
	if srv.hashType == "sha256" {
		hashLength = 64
	}

	if len(input) != hashLength {
		return fmt.Errorf("invalid hash length: %d", len(input))
	}
	if _, err := hex.DecodeString(input); err != nil {
		return err
	}

	ref := reference{
		identity: input,
	}

	// check if the input is already in the gitRefs list
	for _, existingRef := range srv.gitRefs {
		if existingRef.Identity() == input {
			return nil
		}
	}

	srv.lock.Lock()
	srv.gitRefs = append(srv.gitRefs, ref)
	srv.lock.Unlock()

	return nil
}

func (srv *omniBor) addGitRef(reader io.Reader, bom Identifier, length int64) error {
	// add an initial option specifying the length
	options := []gitoid.Option{
		gitoid.WithContentLength(length),
	}

	// populate any options we need
	for _, option := range srv.gitoidOptions {
		options = append(options, option)
	}
	identity, err := gitoid.New(reader, options...)
	if err != nil {
		return err
	}

	ref := reference{
		identity: identity.String(),
		bom:      bom,
	}

	srv.lock.Lock()
	srv.gitRefs = append(srv.gitRefs, ref)
	srv.lock.Unlock()
	return nil
}

func (srv *omniBor) References() []Reference {
	srv.lock.Lock()
	by(referenceSorter).sort(srv.gitRefs)
	result := make([]Reference, 0, len(srv.gitRefs))
	for _, ref := range srv.gitRefs {
		result = append(result, ref)
	}
	srv.lock.Unlock()
	return srv.gitRefs
}

func (srv *omniBor) String() string {
	srv.lock.Lock()
	by(referenceSorter).sort(srv.gitRefs)
	refs := make([]string, 0)
	for _, ref := range srv.gitRefs {
		refs = append(refs, ref.String())
	}
	srv.lock.Unlock()
	return strings.Join(refs, "")
}

func (srv *omniBor) gitRef() string {
	generated := srv.String()
	// add an initial option specifying the length
	options := []gitoid.Option{
		gitoid.WithContentLength(int64(len(generated))),
	}

	// populate any options we need
	for _, option := range srv.gitoidOptions {
		options = append(options, option)
	}

	res, err := gitoid.New(bytes.NewBuffer([]byte(generated)), options...)
	if err != nil {
		// we should only see this if the runtime was fundamentally broken
		panic(err)
	}
	return res.String()
}

func (srv *omniBor) Identity() string {
	return srv.gitRef()
}

type identifier struct {
	identity string
}

func (gb identifier) Identity() string {
	return gb.identity
}

func NewIdentifier(identity string) (Identifier, error) {
	// TODO check if omnibor matches the format
	_, err := hex.DecodeString(identity)
	if err != nil {
		return nil, err
	}
	return &identifier{
		identity: identity,
	}, nil
}
