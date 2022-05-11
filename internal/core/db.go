package core

import (
	"fmt"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/lim-team/LiMaoProxyPool/pkg/util"
)

type IPType string

const (
	Http   IPType = "http"
	Https  IPType = "https"
	Socks4 IPType = "socks4"
	Socks5 IPType = "socks5"
)

type DB interface {
	// 添加IP
	IPAddOrUpdate(ip *IP) error
	// 获取最快最高效的IP
	IPFast(ipTypes ...IPType) (*IP, error)
	// 获取所有有效IP
	IPsVaild() ([]*IP, error)
	// IP有效数量
	IPVaildCount() (int, error)
	// 获取需要检查速度的IP
	IPsNeedCheck(checkExpire time.Duration) ([]*IP, error)
	// 速度更新
	IPSpeedUpdate(ip *IP, speed int) error
	// IP移除
	IPRemove(ip *IP) error

	Load() error  // 加载所有IP到内存中
	Save() error  //  保存所有IP到数据库
	Close() error // 关闭
}

type memoryDB struct {
	ipMap sync.Map
	db    *badger.DB
	opts  *Options
}

func newMemoryDB(opts *Options) *memoryDB {
	var err error
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		panic(err)
	}
	return &memoryDB{
		ipMap: sync.Map{},
		db:    db,
		opts:  opts,
	}
}

func (m *memoryDB) IPAddOrUpdate(ip *IP) error {
	m.ipMap.Store(ip.Key(), ip)
	return nil
}

func (m *memoryDB) IPFast(ipTypes ...IPType) (*IP, error) {
	var fastIP *IP
	m.ipMap.Range(func(key, value interface{}) bool {
		ip := value.(*IP)
		vail := false // ip是否有效
		if len(ipTypes) > 0 {
			for _, ipType := range ipTypes {
				if ipType == ip.Type {
					vail = true
					break
				}
			}
		} else {
			vail = true
		}
		if !vail {
			return true
		}
		if ip.Speed == 0 || ip.Speed >= m.opts.SpeedMax {
			return true
		}
		if ip.LastVisit != 0 && ip.LastVisit > int(time.Now().Unix())-5 {
			return true
		}
		if fastIP == nil && ip.Speed != 0 {
			fastIP = ip
		}
		if ip.Speed != 0 && ip.Speed < fastIP.Speed {
			fastIP = ip
		}
		return true
	})
	if fastIP != nil {
		fastIP.LastVisit = int(time.Now().Unix())
	}
	return fastIP, nil
}

func (m *memoryDB) IPsVaild() ([]*IP, error) {
	ips := make([]*IP, 0)
	m.ipMap.Range(func(key, value interface{}) bool {
		ip := value.(*IP)
		if ip.Speed > 0 && ip.Speed < m.opts.SpeedMax {
			ips = append(ips, ip)
		}

		return true
	})
	return ips, nil
}

func (m *memoryDB) IPsNeedCheck(checkExpire time.Duration) ([]*IP, error) {
	speedCheckIPs := make([]*IP, 0)
	m.ipMap.Range(func(key, value interface{}) bool {
		ip := value.(*IP)
		if ip.LastSpeedAt < int(time.Now().Add(-checkExpire).Unix()) {
			speedCheckIPs = append(speedCheckIPs, ip)
		}
		return true
	})

	return speedCheckIPs, nil
}

func (m *memoryDB) IPSpeedUpdate(ip *IP, speed int) error {
	ip.Speed = speed
	ip.LastSpeedAt = int(time.Now().Unix())
	m.ipMap.Store(ip.Key(), ip)
	return nil
}

func (m *memoryDB) IPVaildCount() (int, error) {
	count := 0
	m.ipMap.Range(func(key, value interface{}) bool {
		ip := value.(*IP)
		if ip.Speed > 0 && ip.Speed < m.opts.SpeedMax {
			count++
		}

		return true
	})
	return count, nil
}

func (m *memoryDB) IPRemove(ip *IP) error {
	m.ipMap.Delete(ip.Key())
	return nil
}
func (m *memoryDB) Load() error {

	var iplist = &IPList{}
	m.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("ips"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			if len(val) == 0 {
				return nil
			}

			return util.ReadJsonByByte(val, &iplist)
		})
	})
	fmt.Println("db加载到IP数量->", len(iplist.IPS))
	if len(iplist.IPS) > 0 {
		for _, ip := range iplist.IPS {
			ip.SpeedChecking = false
			m.IPAddOrUpdate(ip)
		}
	}
	return nil
}

func (m *memoryDB) Save() error {
	ips := make([]*IP, 0)
	m.ipMap.Range(func(key, value interface{}) bool {
		ips = append(ips, value.(*IP))
		return true
	})

	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("ips"), []byte(util.ToJson(&IPList{
			IPS: ips,
		})))
	})
}

func (m *memoryDB) Close() error {
	m.Save()
	return m.db.Close()
}
