package dblentry

import "sort"

import "github.com/tn47/goledger/api"

type DoubleEntry struct {
	name     string
	balances map[string]*Commodity
	credits  map[string]*Commodity
	debits   map[string]*Commodity
}

func newDoubleEntry(name string) *DoubleEntry {
	de := &DoubleEntry{
		name:     name,
		balances: make(map[string]*Commodity),
		credits:  make(map[string]*Commodity),
		debits:   make(map[string]*Commodity),
	}
	return de
}

func (de *DoubleEntry) Balance(obj interface{}) (comm api.Commoditiser) {
	switch v := obj.(type) {
	case *Commodity:
		comm, _ = de.balances[v.name]
	case string:
		comm, _ = de.balances[v]
	}
	return
}

func (de *DoubleEntry) Balances() []api.Commoditiser {
	keys := []string{}
	for name := range de.balances {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	comms := []api.Commoditiser{}
	for _, key := range keys {
		comms = append(comms, de.balances[key])
	}
	return comms
}

func (de *DoubleEntry) Debit(obj interface{}) (comm api.Commoditiser) {
	switch v := obj.(type) {
	case *Commodity:
		comm, _ = de.debits[v.name]
	case string:
		comm, _ = de.debits[v]
	}
	return
}

func (de *DoubleEntry) Debits() []api.Commoditiser {
	keys := []string{}
	for name := range de.debits {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	comms := []api.Commoditiser{}
	for _, key := range keys {
		comms = append(comms, de.debits[key])
	}
	return comms
}

func (de *DoubleEntry) Credit(obj interface{}) (comm api.Commoditiser) {
	switch v := obj.(type) {
	case *Commodity:
		comm, _ = de.credits[v.name]
	case string:
		comm, _ = de.credits[v]
	}
	return
}

func (de *DoubleEntry) Credits() []api.Commoditiser {
	keys := []string{}
	for name := range de.credits {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	comms := []api.Commoditiser{}
	for _, key := range keys {
		comms = append(comms, de.credits[key])
	}
	return comms
}

func (de *DoubleEntry) AddBalance(comm *Commodity) error {
	if balance, ok := de.balances[comm.name]; ok {
		balance.amount += comm.amount
		de.balances[comm.name] = balance
	} else {
		de.balances[comm.name] = comm.makeSimilar(comm.amount)
	}
	// maintain credits and debits.
	if comm.IsDebit() {
		if debit, ok := de.debits[comm.name]; ok {
			debit.amount += comm.amount
			de.debits[comm.name] = debit
		} else {
			de.debits[comm.name] = comm.makeSimilar(comm.amount)
		}
	} else {
		if credit, ok := de.credits[comm.name]; ok {
			credit.amount += -comm.amount // negated
			de.credits[comm.name] = credit
		} else {
			de.credits[comm.name] = comm.makeSimilar(-comm.amount) // negated
		}
	}
	return nil
}

func (de *DoubleEntry) Clone() *DoubleEntry {
	nde := newDoubleEntry(de.name)
	for k, v := range de.balances {
		nde.balances[k] = v
	}
	for k, v := range de.credits {
		nde.credits[k] = v
	}
	for k, v := range de.debits {
		nde.debits[k] = v
	}
	return nde
}
