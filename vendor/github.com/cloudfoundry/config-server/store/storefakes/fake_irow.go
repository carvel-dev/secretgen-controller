// Code generated by counterfeiter. DO NOT EDIT.
package storefakes

import (
	"sync"

	"github.com/cloudfoundry/config-server/store"
)

type FakeIRow struct {
	ScanStub        func(dest ...interface{}) error
	scanMutex       sync.RWMutex
	scanArgsForCall []struct {
		dest []interface{}
	}
	scanReturns struct {
		result1 error
	}
	scanReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIRow) Scan(dest ...interface{}) error {
	fake.scanMutex.Lock()
	ret, specificReturn := fake.scanReturnsOnCall[len(fake.scanArgsForCall)]
	fake.scanArgsForCall = append(fake.scanArgsForCall, struct {
		dest []interface{}
	}{dest})
	fake.recordInvocation("Scan", []interface{}{dest})
	fake.scanMutex.Unlock()
	if fake.ScanStub != nil {
		return fake.ScanStub(dest...)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.scanReturns.result1
}

func (fake *FakeIRow) ScanCallCount() int {
	fake.scanMutex.RLock()
	defer fake.scanMutex.RUnlock()
	return len(fake.scanArgsForCall)
}

func (fake *FakeIRow) ScanArgsForCall(i int) []interface{} {
	fake.scanMutex.RLock()
	defer fake.scanMutex.RUnlock()
	return fake.scanArgsForCall[i].dest
}

func (fake *FakeIRow) ScanReturns(result1 error) {
	fake.ScanStub = nil
	fake.scanReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIRow) ScanReturnsOnCall(i int, result1 error) {
	fake.ScanStub = nil
	if fake.scanReturnsOnCall == nil {
		fake.scanReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.scanReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeIRow) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.scanMutex.RLock()
	defer fake.scanMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIRow) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ store.IRow = new(FakeIRow)
