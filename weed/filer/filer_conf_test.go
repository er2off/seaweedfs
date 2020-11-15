package filer

import (
	"testing"

	"github.com/chrislusf/seaweedfs/weed/pb/filer_pb"
	"github.com/stretchr/testify/assert"
	"github.com/viant/ptrie"
)

func TestFilerConf(t *testing.T) {

	fc := &FilerConf{
		rules: ptrie.New(),
	}
	conf := &filer_pb.FilerConf{Locations: []*filer_pb.FilerConf_PathConf{
		{
			LocationPrefix: "/buckets/abc",
			Collection:     "abc",
		},
		{
			LocationPrefix: "/buckets/abcd",
			Collection:     "abcd",
		},
	}}
	fc.doLoadConf(conf)

	assert.Equal(t, "abc", fc.MatchStorageRule("/buckets/abc/jasdf").Collection)
	assert.Equal(t, "abcd", fc.MatchStorageRule("/buckets/abcd/jasdf").Collection)

}
