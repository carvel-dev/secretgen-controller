// Code generated by counterfeiter. DO NOT EDIT.
package storefakes

import (
	"sync"

	"github.com/BurntSushi/migration"
	"github.com/cloudfoundry/config-server/store"
)

type FakeISql struct {
	OpenWithStub        func(driverName, dataSourceName string, migrations []migration.Migrator, getVersion migration.GetVersion, setVersion migration.SetVersion) (store.IDb, error)
	openWithMutex       sync.RWMutex
	openWithArgsForCall []struct {
		driverName     string
		dataSourceName string
		migrations     []migration.Migrator
		getVersion     migration.GetVersion
		setVersion     migration.SetVersion
	}
	openWithReturns struct {
		result1 store.IDb
		result2 error
	}
	openWithReturnsOnCall map[int]struct {
		result1 store.IDb
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeISql) OpenWith(driverName string, dataSourceName string, migrations []migration.Migrator, getVersion migration.GetVersion, setVersion migration.SetVersion) (store.IDb, error) {
	var migrationsCopy []migration.Migrator
	if migrations != nil {
		migrationsCopy = make([]migration.Migrator, len(migrations))
		copy(migrationsCopy, migrations)
	}
	fake.openWithMutex.Lock()
	ret, specificReturn := fake.openWithReturnsOnCall[len(fake.openWithArgsForCall)]
	fake.openWithArgsForCall = append(fake.openWithArgsForCall, struct {
		driverName     string
		dataSourceName string
		migrations     []migration.Migrator
		getVersion     migration.GetVersion
		setVersion     migration.SetVersion
	}{driverName, dataSourceName, migrationsCopy, getVersion, setVersion})
	fake.recordInvocation("OpenWith", []interface{}{driverName, dataSourceName, migrationsCopy, getVersion, setVersion})
	fake.openWithMutex.Unlock()
	if fake.OpenWithStub != nil {
		return fake.OpenWithStub(driverName, dataSourceName, migrations, getVersion, setVersion)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.openWithReturns.result1, fake.openWithReturns.result2
}

func (fake *FakeISql) OpenWithCallCount() int {
	fake.openWithMutex.RLock()
	defer fake.openWithMutex.RUnlock()
	return len(fake.openWithArgsForCall)
}

func (fake *FakeISql) OpenWithArgsForCall(i int) (string, string, []migration.Migrator, migration.GetVersion, migration.SetVersion) {
	fake.openWithMutex.RLock()
	defer fake.openWithMutex.RUnlock()
	return fake.openWithArgsForCall[i].driverName, fake.openWithArgsForCall[i].dataSourceName, fake.openWithArgsForCall[i].migrations, fake.openWithArgsForCall[i].getVersion, fake.openWithArgsForCall[i].setVersion
}

func (fake *FakeISql) OpenWithReturns(result1 store.IDb, result2 error) {
	fake.OpenWithStub = nil
	fake.openWithReturns = struct {
		result1 store.IDb
		result2 error
	}{result1, result2}
}

func (fake *FakeISql) OpenWithReturnsOnCall(i int, result1 store.IDb, result2 error) {
	fake.OpenWithStub = nil
	if fake.openWithReturnsOnCall == nil {
		fake.openWithReturnsOnCall = make(map[int]struct {
			result1 store.IDb
			result2 error
		})
	}
	fake.openWithReturnsOnCall[i] = struct {
		result1 store.IDb
		result2 error
	}{result1, result2}
}

func (fake *FakeISql) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.openWithMutex.RLock()
	defer fake.openWithMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeISql) recordInvocation(key string, args []interface{}) {
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

var _ store.ISql = new(FakeISql)
