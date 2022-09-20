package example

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/smt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"
)

const (
	URI = "mongodb://127.0.0.1:27017/?gssapiServiceName=mongodb"
)

func initMongoSmtTree(num, count int) ([]*smt.SparseMerkleTree, error) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI))
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect err: %s", err.Error())
	}

	var list []*smt.SparseMerkleTree
	for i := 0; i < num; i++ {
		store := smt.NewMongoStore(ctx, client, "testsmt", fmt.Sprintf("test-%d", i))
		tree := smt.NewSparseMerkleTree(store)
		for j := 0; j < count; j++ {
			fmt.Println(i, "-", j)
			key := fmt.Sprintf("key-%d", j)
			value := fmt.Sprintf("value-%d", j)

			k := smt.Sha256(key)
			v := smt.Sha256(value)
			if err := tree.Update(k, v); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
		}
		list = append(list, tree)
	}
	return list, nil
}

func TestInitSmt(t *testing.T) {
	num := 10
	_, err := initMongoSmtTree(num, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSmt(t *testing.T) {
	num := 1
	list, err := initMongoSmtTree(num, 1)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		http.ListenAndServe(":8899", nil)
	}()
	wg := sync.WaitGroup{}

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := buildSmt(index, list[index]); err != nil {
				panic(err)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("ok")

	select {}
}

func buildSmt(i int, tree *smt.SparseMerkleTree) error {
	for i := 0; i < 100; i++ {
		if _, err := tree.Root(); err != nil {
			return fmt.Errorf("tree.Root 1 err: %s", err.Error())
		}

		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)

		k := smt.Sha256(key)
		v := smt.Sha256(value)
		if err := tree.Update(k, v); err != nil {
			return fmt.Errorf("tree.Update err: %s", err.Error())
		}

		var keys, values []smt.H256
		keys = append(keys, k)
		values = append(values, v)
		if _, err := tree.MerkleProof(keys, values); err != nil {
			return fmt.Errorf("tree.MerkleProof err: %s", err.Error())
		}

		if _, err := tree.Root(); err != nil {
			return fmt.Errorf("tree.Root 1 err: %s", err.Error())
		}
	}
	fmt.Println("buildSmt:", i)
	return nil
}
