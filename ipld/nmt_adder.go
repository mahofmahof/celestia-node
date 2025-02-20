package ipld

import (
	"context"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-merkledag"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"

	"github.com/celestiaorg/celestia-node/ipld/plugin"
)

// NmtNodeAdder adds ipld.Nodes to the underlying ipld.Batch if it is inserted
// into a nmt tree.
type NmtNodeAdder struct {
	ctx    context.Context
	add    ipld.NodeAdder
	leaves *cid.Set
	err    error
}

// NewNmtNodeAdder returns a new NmtNodeAdder with the provided context and
// batch. Note that the context provided should have a timeout
// It is not thread-safe.
func NewNmtNodeAdder(ctx context.Context, bs blockservice.BlockService, opts ...ipld.BatchOption) *NmtNodeAdder {
	return &NmtNodeAdder{
		add:    ipld.NewBatch(ctx, merkledag.NewDAGService(bs), opts...),
		ctx:    ctx,
		leaves: cid.NewSet(),
	}
}

// Visit is a NodeVisitor that can be used during the creation of a new NMT to
// create and add ipld.Nodes to the Batch while computing the root of the NMT.
func (n *NmtNodeAdder) Visit(hash []byte, children ...[]byte) {
	if n.err != nil {
		return // protect from further visits if there is an error
	}

	id := plugin.MustCidFromNamespacedSha256(hash)
	switch len(children) {
	case 1:
		if n.leaves.Visit(id) {
			n.err = n.add.Add(n.ctx, plugin.NewNMTLeafNode(id, children[0]))
		}
	case 2:
		n.err = n.add.Add(n.ctx, plugin.NewNMTNode(id, children[0], children[1]))
	default:
		panic("expected a binary tree")
	}
}

// NodeAdder returns the ipld.NodeAdder originally provided to the NmtNodeAdder.
func (n *NmtNodeAdder) NodeAdder() ipld.NodeAdder {
	return n.add
}

// Commit checks for errors happened during Visit and if absent commits data to inner Batch.
func (n *NmtNodeAdder) Commit() error {
	if n.err != nil {
		return n.err
	}

	if b, ok := n.add.(*ipld.Batch); ok {
		return b.Commit()
	}
	return nil
}
