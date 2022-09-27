package example

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/smt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"
)

const (
	URI = "mongodb://127.0.0.1:27017/?gssapiServiceName=mongodb"
)

// go test -v -timeout=0 -run TestSmt example/smt_test.go
func TestSmt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	num := 100
	buildNumStart := 0
	buildNumEnd := 100
	list, err := initMongoSmtTree(ctx, num, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	//go func() {
	//	http.ListenAndServe(":8899", nil)
	//}()
	wg := sync.WaitGroup{}
	now := time.Now()
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), len(list))
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := buildSmt(index, list[index], buildNumStart, buildNumEnd); err != nil {
				t.Error("buildSmt:", err.Error())
			}
		}(i)
	}
	wg.Wait()
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("ok", time.Since(now).Minutes())

	cancel()
	//select {}
}

func initMongoSmtTree(ctx context.Context, num, countStart, countEnd int) ([]*smt.SparseMerkleTree, error) {

	var list []*smt.SparseMerkleTree
	for i := 0; i < num; i++ {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI))
		if err != nil {
			return nil, fmt.Errorf("mongo.Connect err: %s", err.Error())
		}
		store := smt.NewMongoStore(ctx, client, "testsmt", fmt.Sprintf("test-%d", i))
		tree := smt.NewSparseMerkleTree(store)
		for j := countStart; j < countEnd; j++ {
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
	num := 100
	ctx, cancel := context.WithCancel(context.Background())
	_, err := initMongoSmtTree(ctx, num, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
}

func buildSmt(j int, tree *smt.SparseMerkleTree, buildNumStart, buildNumEnd int) error {
	for i := buildNumStart; i < buildNumEnd; i++ {
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
		fmt.Println("buildSmt :", j, "-", i)
	}
	fmt.Println("buildSmt OK:", j)
	return nil
}
