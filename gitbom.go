package gitbom

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"sort"
	"strings"
)

type GitRef interface {
	Identity() string
	Bom() OpaqueGitBom
	String() string
}

type gitRef struct {
	identity string
	bom      OpaqueGitBom
}

type gitRefSorter struct {
	refs []GitRef
	by   By
}
type By func(p1, p2 GitRef) bool

func GitRefSorter(r1, r2 GitRef) bool {
	return r1.Identity() < r2.Identity()
}

func (by By) Sort(refs []GitRef) {
	sorter := &gitRefSorter{
		refs: refs,
		by:   by,
	}
	sort.Sort(sorter)
}

func (grs *gitRefSorter) Len() int {
	return len(grs.refs)
}

func (grs *gitRefSorter) Swap(i, j int) {
	grs.refs[i], grs.refs[j] = grs.refs[j], grs.refs[i]
}

func (grs *gitRefSorter) Less(i, j int) bool {
	return grs.by(grs.refs[i], grs.refs[j])
}

func (ref gitRef) Identity() string {
	return ref.identity
}

func (ref gitRef) Bom() OpaqueGitBom {
	return ref.Bom()
}

func (ref gitRef) String() string {
	res := fmt.Sprintf("blob %s", ref.identity)
	if ref.bom != nil {
		res = fmt.Sprintf("%s bom %s", res, ref.bom.Identity())
	}

	res = res + "\n"
	return res
}

type OpaqueGitBom interface {
	Identity() string
}

type GitBom interface {
	OpaqueGitBom
	AddSha1GitRef(obj []byte, bom OpaqueGitBom) error
	AddSha1GitRefFromReader(reader io.Reader, bom OpaqueGitBom, objLength int64) error
	AddSha256GitRef(obj []byte, bom OpaqueGitBom) error
	AddSha256GitRefFromReader(reader io.Reader, bom OpaqueGitBom, objLength int64) error
	GitRefs() []GitRef
	Sha1GitRef() string
	Sha256GitRef() string
	String() string
}

type gitBom struct {
	gitRefs []GitRef
}

// NewGitBom creates a new GitBom object.
// Thread Safety: none, apply your own controls.
//
// Adding duplicate objects with the same GitRef identity results in only one GitRef entry.
// GitRefs are sorted in ascending order based on their UTF-8 values.
//
// Implementation details:
// Adding a GitRef is O(n) to discover duplicates.
// Generating a GitBom is O(n*log(n)) as it sorts the existing refs.
func NewGitBom() GitBom {
	return &gitBom{}
}

func (srv *gitBom) AddSha1GitRef(obj []byte, bom OpaqueGitBom) error {
	hashAlgorithm := sha1.New()
	reader := bytes.NewBuffer(obj)
	return srv.addGitRef(reader, bom, hashAlgorithm, int64(len(obj)))
}

func (srv *gitBom) AddSha256GitRef(obj []byte, bom OpaqueGitBom) error {
	hashAlgorithm := sha256.New()
	reader := bytes.NewBuffer(obj)
	return srv.addGitRef(reader, bom, hashAlgorithm, int64(len(obj)))
}

func (srv *gitBom) AddSha1GitRefFromReader(reader io.Reader, bom OpaqueGitBom, objLength int64) error {
	hashAlgorithm := sha1.New()
	return srv.addGitRef(reader, bom, hashAlgorithm, objLength)
}

func (srv *gitBom) AddSha256GitRefFromReader(reader io.Reader, bom OpaqueGitBom, objLength int64) error {
	hashAlgorithm := sha256.New()
	return srv.addGitRef(reader, bom, hashAlgorithm, objLength)
}

func (srv *gitBom) addGitRef(reader io.Reader, bom OpaqueGitBom, hashAlgorithm hash.Hash, length int64) error {
	identity, err := generateGitHash(reader, hashAlgorithm, length)
	if err != nil {
		return err
	}

	// refs may be unsorted
	for _, cur := range srv.gitRefs {
		if identity == cur.Identity() {
			// we found the object in gitrefs, return
			return nil
		}
	}

	ref := gitRef{
		identity: identity,
		bom:      bom,
	}

	srv.gitRefs = append(srv.gitRefs, ref)
	return nil
}

func (srv *gitBom) GitRefs() []GitRef {
	By(GitRefSorter).Sort(srv.gitRefs)
	return srv.gitRefs
}

func (srv *gitBom) String() string {
	By(GitRefSorter).Sort(srv.gitRefs)
	refs := make([]string, 0)
	for _, ref := range srv.gitRefs {
		refs = append(refs, ref.String())
	}
	return strings.Join(refs, "")
}

func (srv *gitBom) Sha1GitRef() string {
	generated := srv.String()
	hashAlgorithm := sha1.New()
	res, err := generateGitHash(bytes.NewBuffer([]byte(generated)), hashAlgorithm, int64(len(generated)))
	if err != nil {
		// we should only see this if the runtime was fundamentally broken
		panic(err)
	}
	return res
}

func (srv *gitBom) Sha256GitRef() string {
	generated := srv.String()
	hashAlgorithm := sha256.New()
	res, err := generateGitHash(bytes.NewBuffer([]byte(generated)), hashAlgorithm, int64(len(generated)))
	if err != nil {
		// we should only see this if the runtime was fundamentally broken
		panic(err)
	}
	return res
}

func (srv *gitBom) Identity() string {
	return srv.Sha256GitRef()
}

func generateGitHash(reader io.Reader, hashAlgorithm hash.Hash, length int64) (string, error) {
	// \u0000 is the unicode sequence for '\0'
	header := fmt.Sprintf("blob %d\u0000", length)
	n, err := hashAlgorithm.Write([]byte(header))
	if err != nil {
		return "", err
	}
	if n != len(header) {
		return "", errors.New("impartial header write while generating git ref")
	}

	written, err := io.Copy(hashAlgorithm, reader)
	if err != nil {
		return "", err
	}
	if written != length {
		return "", errors.New("actual reader length did not match expected length")
	}

	hashBytes := hashAlgorithm.Sum([]byte{})
	hashStr := hex.EncodeToString(hashBytes)
	return hashStr, nil
}

type opaqueGitBom struct {
	identity string
}

func (gb opaqueGitBom) Identity() string {
	return gb.identity
}

func NewOpaqueGitBom(identity string) (OpaqueGitBom, error) {
	// TODO check if it matches the format
	_, err := hex.DecodeString(identity)
	if err != nil {
		return nil, err
	}
	return &opaqueGitBom{
		identity: identity,
	}, nil
}
