package ch_pool

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"sync"
	"time"
)

var (
	errInvalidConfig = errors.New("invalid pool config")
	errPoolClosed    = errors.New("pool closed")
)

type factory func()(*sqlx.DB, error)

type Pool interface {
	GetConn() (*sqlx.DB, error) 	// get connect resource
	Release(*sqlx.DB) error     	// release connect resource
	Close(*sqlx.DB) error       	// close connect resource
	Shutdown() error             	// close pool
}

type ChPool struct {
	sync.Mutex
	pool        chan *sqlx.DB
	maxOpen     int  // max open connect num
	numOpen     int  // now pool has open connect num
	minOpen     int  // min open connect num
	closed      bool // pool is close
	maxLifetime time.Duration // connect max life time
	factory     factory // the factory of create connect
}

func NewChPool(minOpen, maxOpen int, maxLifetime time.Duration, factory factory) (*ChPool, error) {
	if maxOpen <= 0 || minOpen > maxOpen {
		return nil, errInvalidConfig
	}
	p := &ChPool{
		maxOpen:     maxOpen,
		minOpen:     minOpen,
		maxLifetime: maxLifetime,
		factory:     factory,
		pool:        make(chan *sqlx.DB, maxOpen),
	}

	for i := 0; i < minOpen; i++ {
		dbConn, err := factory()
		if err != nil {
			continue
		}
		p.numOpen++
		p.pool <- dbConn
	}
	return p, nil
}

func (p *ChPool) GetConn()(dbConn *sqlx.DB, err error) {
	if p.closed {
		err = errPoolClosed
		return
	}
	for {
		dbConn, err = p.getOrCreate()
		// todo max Lift time处理
		return
	}
}

func (p *ChPool) getOrCreate()(dbConn *sqlx.DB, err error) {
	select {
	case dbConn = <-p.pool:
		return
	default:
	}
	p.Lock()
	if p.numOpen >= p.maxOpen {
		dbConn = <-p.pool
		p.Unlock()
		return
	}
	// 新建连接
	dbConn, err = p.factory()
	if err != nil {
		p.Unlock()
		return
	}
	p.numOpen++
	p.Unlock()
	return
}

// 释放单个资源到连接池
func (p *ChPool) Release(dbConn *sqlx.DB) error {
	if p.closed {
		return errPoolClosed
	}
	if p.numOpen > p.minOpen {
		err := p.Close(dbConn)
		return err
	}
	p.Lock()
	p.pool <- dbConn
	p.Unlock()
	return nil
}

// 关闭单个资源
func (p *ChPool) Close(dbConn *sqlx.DB) error {
	p.Lock()
	err := dbConn.Close()
	p.numOpen--
	p.Unlock()
	return err
}

// 关闭连接池，释放所有资源
func (p *ChPool) Shutdown() error {
	if p.closed {
		return errPoolClosed
	}
	p.Lock()
	close(p.pool)
	for dbConn := range p.pool {
		_ = dbConn.Close()
		p.numOpen--
	}
	p.closed = true
	p.Unlock()
	return nil
}