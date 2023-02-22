package util

import (
	"crypto/rand"
	"crypto/sha256"

	"github.com/ipfs/go-cid"
	u "github.com/ipfs/go-ipfs-util"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
)

// Content encapsulates multiple representations of the same data.
type Content struct {
	Raw   []byte
	mhash mh.Multihash
	CID   cid.Cid
}

// NewRandomContent reads 1024 bytes from crypto/rand and builds a content struct.
func NewRandomContent() (*Content, error) {
	raw := make([]byte, 1024)
	if _, err := rand.Read(raw); err != nil {
		return nil, errors.Wrap(err, "read rand data")
	}
	hash := sha256.New()
	hash.Write(raw)

	mhash, err := mh.Encode(hash.Sum(nil), mh.SHA2_256)
	if err != nil {
		return nil, errors.Wrap(err, "encode multi hash")
	}

	return &Content{
		Raw:   raw,
		mhash: mhash,
		CID:   cid.NewCidV0(mhash),
	}, nil
}

// ContentFrom takes the given bytes and builds a content struct.
func ContentFrom(raw []byte) (*Content, error) {
	hash := sha256.New()
	hash.Write(raw)

	mhash, err := mh.Encode(hash.Sum(nil), mh.SHA2_256)
	if err != nil {
		return nil, errors.Wrap(err, "encode multi hash")
	}

	return &Content{
		Raw:   raw,
		mhash: mhash,
		CID:   cid.NewCidV0(mhash),
	}, nil
}

// DistanceTo returns the XOR distance of the content to the provided peer ID
// as it is used in the libp2p Kademlia DHT.
func (c *Content) DistanceTo(peerID peer.ID) []byte {
	return u.XOR(kbucket.ConvertPeerID(peerID), kbucket.ConvertKey(string(c.mhash)))
}
