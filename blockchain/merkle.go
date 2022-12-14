package blockchain

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode

	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, dat := range data {
		node := NewMerkleNode(nil, nil, dat)
		nodes = append(nodes, node)
	}

	for i := 0; i < len(data)/2; i++ {
		var level []*MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(nodes[j], nodes[j+1], nil)
			level = append(level, node)
		}

		nodes = level
	}

	tree := &MerkleTree{
		RootNode: nodes[0],
	}

	return tree
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	var hash [32]byte
	if left == nil && right == nil {
		hash = sha256.Sum256(data)
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash = sha256.Sum256(prevHashes)
	}
	node.Data = hash[:]
	node.Left = left
	node.Right = right

	return &node
}
