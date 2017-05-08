package dblentry

import "fmt"
import "sort"
import "strings"
import "time"

import "github.com/prataprc/goledger/api"
import "github.com/prataprc/golog"

type Datastore struct {
	name        string
	reporter    api.Reporter
	transdb     *DB
	pricedb     *DB
	accntdb     map[string]*Account // full account-name -> account
	balance     map[string]*Commodity
	defaultcomm string
	commodities map[string]*Commodity
	// directive fields
	currdate     time.Time
	aliases      map[string]string // alias, account-alias
	payees       map[string]string // account-payee map[regex]->accountname
	rootaccount  string            // apply-account
	blncingaccnt string            // account
}

func NewDatastore(name string, reporter api.Reporter) *Datastore {
	db := &Datastore{
		name:        name,
		reporter:    reporter,
		transdb:     NewDB(fmt.Sprintf("%v-transactions", name)),
		pricedb:     NewDB(fmt.Sprintf("%v-pricedb", name)),
		accntdb:     map[string]*Account{},
		balance:     make(map[string]*Commodity),
		commodities: map[string]*Commodity{},
		// directives
		currdate: time.Now(),
		aliases:  map[string]string{},
	}
	db.defaultprices()
	return db
}

//---- accessor

func (db *Datastore) GetCommodity(name string, defcomm *Commodity) *Commodity {
	if name == "" && db.defaultcomm != "" {
		return db.commodities[db.defaultcomm]
	}
	if db.defaultcomm == "" && name != "" {
		db.defaultcomm = name
	}
	if comm, ok := db.commodities[name]; ok {
		return comm
	}
	log.Debugf("commodity %q added\n", name)
	if defcomm == nil {
		defcomm = NewCommodity(name)
	}
	db.commodities[name] = defcomm
	return defcomm
}

func (db *Datastore) GetAccount(name string) *Account {
	if name == "" {
		return nil
	}
	account, ok := db.accntdb[name]
	if ok == false {
		account = NewAccount(name)
	}
	db.accntdb[name] = account
	return account
}

func (db *Datastore) Accountnames() []string {
	accnames := []string{}
	for name := range db.accntdb {
		accnames = append(accnames, name)
	}
	return accnames
}

func (db *Datastore) HasAccount(name string) bool {
	_, ok := db.accntdb[name]
	return ok
}

func (db *Datastore) Balance(obj interface{}) (balance api.Commoditiser) {
	switch v := obj.(type) {
	case *Commodity:
		balance, _ = db.balance[v.name]
	case string:
		balance, _ = db.balance[v]
	}
	return balance
}

func (db *Datastore) Balances() []api.Commoditiser {
	keys := []string{}
	for name := range db.balance {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	comms := []api.Commoditiser{}
	for _, key := range keys {
		comms = append(comms, db.balance[key])
	}
	return comms
}

func (db *Datastore) SubAccounts(parentname string) []*Account {
	accounts := []*Account{}
	for name, account := range db.accntdb {
		if strings.HasPrefix(parentname, name) {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

func (db *Datastore) SetYear(year int) *Datastore {
	db.currdate = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	return db
}

func (db *Datastore) Year() int {
	return db.currdate.Year()
}

func (db *Datastore) SetCurrentDate(date time.Time) *Datastore {
	db.currdate = date
	return db
}

func (db *Datastore) CurrentDate() time.Time {
	return db.currdate
}

//---- engine

func (db *Datastore) Firstpass(obj interface{}) error {
	if trans, ok := obj.(*Transaction); ok {
		if err := trans.Firstpass(db); err != nil {
			return err
		}
		db.SetCurrentDate(trans.date)
		db.transdb.Insert(trans.date, trans)
		return nil

	} else if price, ok := obj.(*Price); ok {
		return db.pricedb.Insert(price.when, price)

	} else if directive, ok := obj.(*Directive); ok {
		if err := directive.Firstpass(db); err != nil {
			return err
		}
		return nil
	}
	panic("unreachable code")
}

func (db *Datastore) Secondpass() error {
	kvfull := make([]KV, 0)
	for _, kv := range db.transdb.Range(nil, nil, "both", kvfull) {
		trans := kv.v.(*Transaction)
		if err := trans.Secondpass(db); err != nil {
			return err
		}
	}
	return nil
}

func (db *Datastore) AddBalance(commodity *Commodity) {
	if balance, ok := db.balance[commodity.name]; ok {
		balance.amount += commodity.amount
		db.balance[commodity.name] = balance
		return
	}
	db.balance[commodity.name] = commodity.Similar(commodity.amount)
}

func (db *Datastore) DeductBalance(commodity *Commodity) {
	if balance, ok := db.balance[commodity.name]; ok {
		balance.amount -= commodity.amount
		db.balance[commodity.name] = balance
		return
	}
	db.balance[commodity.name] = commodity.Similar(commodity.amount)
}

// directive-alias

func (db *Datastore) AddAlias(aliasname, accountname string) *Datastore {
	db.aliases[aliasname] = accountname
	return db
}

func (db *Datastore) GetAlias(aliasname string) (accountname string, ok bool) {
	accountname, ok = db.aliases[aliasname]
	return accountname, ok
}

// directive-apply-account

func (db *Datastore) SetRootaccount(name string) *Datastore {
	db.rootaccount = name
	return db
}

func (db *Datastore) Rootaccount() string {
	return db.rootaccount
}

// directive-account

func (db *Datastore) Declare(value interface{}) error {
	switch v := value.(type) {
	case *Account:
		account := db.GetAccount(v.name)
		account.SetDirective(v)
		if v.defblns {
			db.SetBalancingaccount(v.name)
		}
		return nil

	default:
		panic("unreachable code")
	}
	panic("unreachable code")
}

func (db *Datastore) AddPayee(regex, accountname string) *Datastore {
	db.payees[regex] = accountname
	return db
}

func (db *Datastore) SetBalancingaccount(name string) *Datastore {
	db.blncingaccnt = name
	return db
}

func (db *Datastore) LookupAlias(name string) string {
	if accountname, ok := db.aliases[name]; ok {
		return accountname
	}
	return name
}

func (db *Datastore) Applyroot(name string) string {
	if db.rootaccount != "" {
		return db.rootaccount + ":" + name
	}
	return name
}

func (db *Datastore) defaultprices() {
	_ = []string{
		"P 01/01/2000 kb 1024b",
		"P 01/01/2000 mb 1024kb",
		"P 01/01/2000 gb 1024mb",
		"P 01/01/2000 tb 1024gb",
		"P 01/01/2000 pb 1024tb",

		"P 01/01/2000 m 60s",
		"P 01/01/2000 h 60m",
	}
}
