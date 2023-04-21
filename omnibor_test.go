package omnibor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/edwarnicke/gitoid"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO create sha256 version
func TestFlatWorkflowSha1(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha1OmniBOR()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\n" +
		"blob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"
	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	assert.NoError(t, err)

	assert.Equal(t, "dc0be356e8c2ba26e66448d97db76ad050206574", ref)
}

func TestFlatWorkflowSha256(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha256OmniBOR()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 8aec4e4876f854f688d0ebfc8f37598f38e5fd6903cccc850ca36591175aeb60\n" +
		"blob 8df3dab4ddfa6eb2a34065cda27d95af2709d4d2658e1b5fbd145822acf42b28\n"
	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	assert.NoError(t, err)

	assert.Equal(t, "e32e7e7761709be17ef573556a82960d489ddf0092424f7db1c91d8363dde822", ref)
}

// TODO add sha256
func TestNestedWorkflowSha1(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha1OmniBOR()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\nblob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"

	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	expected = "dc0be356e8c2ba26e66448d97db76ad050206574"

	assert.Equal(t, expected, ref)

	string3 := "hello2"
	string4 := "independent"
	string5 := "opaque"

	gb2 := NewSha1OmniBOR()

	err = gb2.AddReference([]byte(string3), gb)
	assert.NoError(t, err)

	err = gb2.AddReference([]byte(string4), nil)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom dc0be356e8c2ba26e66448d97db76ad050206574\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())

	identifier, err := NewIdentifier("a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812")
	assert.NoError(t, err)
	err = gb2.AddReference([]byte(string5), identifier)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom dc0be356e8c2ba26e66448d97db76ad050206574\nblob 32898208a218272b0fa7549f60951d4eed2ed830 bom a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())
}

func TestMixedNestedWorkflow(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha256OmniBOR()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 8aec4e4876f854f688d0ebfc8f37598f38e5fd6903cccc850ca36591175aeb60\nblob 8df3dab4ddfa6eb2a34065cda27d95af2709d4d2658e1b5fbd145822acf42b28\n"

	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	expected = "e32e7e7761709be17ef573556a82960d489ddf0092424f7db1c91d8363dde822"

	assert.Equal(t, expected, ref)

	string3 := "hello2"
	string4 := "independent"
	string5 := "opaque"

	gb2 := NewSha1OmniBOR()

	err = gb2.AddReference([]byte(string3), gb)
	assert.NoError(t, err)

	err = gb2.AddReference([]byte(string4), nil)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom e32e7e7761709be17ef573556a82960d489ddf0092424f7db1c91d8363dde822\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())

	identifier, err := NewIdentifier("a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812")
	assert.NoError(t, err)
	err = gb2.AddReference([]byte(string5), identifier)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom e32e7e7761709be17ef573556a82960d489ddf0092424f7db1c91d8363dde822\nblob 32898208a218272b0fa7549f60951d4eed2ed830 bom a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())
}

func TestValidIdentifier(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dd")
	assert.NoError(t, err)
}

func TestInvalidIdentifier_TooFewCharacters(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287d")
	assert.Error(t, err)
}

func TestInvalidIdentifier_NonHexCharacter(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dg")
	assert.Error(t, err)
}

func TestInvalidIdentifier_ExtraTrailingSpace(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidIdentifier_ExtraSpaces(t *testing.T) {
	_, err := NewIdentifier(" 23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidIdentifier_VeryInvalid(t *testing.T) {
	_, err := NewIdentifier(" 23294b0610492cf 55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestAddingExistingReferenceSha1(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gid1, _ := gitoid.New(bytes.NewBufferString(string1))
	gid2, _ := gitoid.New(bytes.NewBufferString(string2))

	gb := NewSha1OmniBOR()
	err := gb.AddExistingReference(gid1.String())
	assert.NoError(t, err)
	err = gb.AddExistingReference(gid2.String())
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\nblob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"

	assert.Equal(t, expected, gb.String())
}

func TestAddingExistingReferenceSha256(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gid1, _ := gitoid.New(bytes.NewBufferString(string1), gitoid.WithSha256())
	gid2, _ := gitoid.New(bytes.NewBufferString(string2), gitoid.WithSha256())

	gb := NewSha256OmniBOR()
	err := gb.AddExistingReference(gid1.String())
	assert.NoError(t, err)
	err = gb.AddExistingReference(gid2.String())
	assert.NoError(t, err)
	expected := "blob 8aec4e4876f854f688d0ebfc8f37598f38e5fd6903cccc850ca36591175aeb60\nblob 8df3dab4ddfa6eb2a34065cda27d95af2709d4d2658e1b5fbd145822acf42b28\n"

	assert.Equal(t, expected, gb.String())
}

func TestAddExistingMalformedSha1(t *testing.T) {
	string1 := "hello"

	gid1, _ := gitoid.New(bytes.NewBufferString(string1))

	gb := NewSha1OmniBOR()
	err := gb.AddExistingReference(gid1.String()[1:])
	assert.Error(t, err)

	malformedHash := gid1.String()
	malformedHash = "g" + malformedHash[1:]
	err = gb.AddExistingReference(malformedHash)
	assert.Error(t, err)

	gid256, _ := gitoid.New(bytes.NewBufferString(string1), gitoid.WithSha256())
	err = gb.AddExistingReference(gid256.String())
	assert.Error(t, err)
}

func TestAddExistingMalformedSha256(t *testing.T) {
	string1 := "hello"

	gid1, _ := gitoid.New(bytes.NewBufferString(string1), gitoid.WithSha256())

	gb := NewSha256OmniBOR()
	err := gb.AddExistingReference(gid1.String()[1:])
	assert.Error(t, err)

	malformedHash := gid1.String()
	malformedHash = "g" + malformedHash[1:]
	err = gb.AddExistingReference(malformedHash)
	assert.Error(t, err)

	gidsha1, _ := gitoid.New(bytes.NewBufferString(string1))
	err = gb.AddExistingReference(gidsha1.String())
	assert.Error(t, err)
}

func BenchmarkNewOmniBOR(b *testing.B) {
	dataset := generateDataset(b.N)

	gb := NewSha1OmniBOR()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//fmt.Println(dataset[i])
		_ = gb.AddReference(dataset[i], nil)
	}
	b.StopTimer()

	fmt.Println(len(gb.References()), len(dataset), b.N)
}

func generateDataset(n int) [][]byte {
	dataset := make([][]byte, 0)
	for i := 0; i < n; i++ {
		buf := make([]byte, 64)
		binary.LittleEndian.PutUint32(buf, uint32(i))
		//fmt.Println(buf)
		dataset = append(dataset, buf)
	}
	for i := 0; i < len(dataset); i++ {
		//fmt.Println(dataset[i])
	}
	return dataset
}
