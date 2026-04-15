package scheduler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/my-llm-api/config"
)

type AccountStatus int

const (
	AccountStatusHealthy AccountStatus = iota
	AccountStatusUnhealthy
)

type Account struct {
	ID         string
	APIKey     string
	Weight     int
	Status     AccountStatus
	LastUsed   time.Time
	ErrorCount int
}

type AccountPool struct {
	accounts    []*Account
	index       uint64
	mu          sync.RWMutex
	maxErrors   int
	recoverTime time.Duration
}

func NewAccountPool(accounts []config.AccountConfig) *AccountPool {
	pool := &AccountPool{
		accounts:    make([]*Account, 0, len(accounts)),
		maxErrors:   3,
		recoverTime: 5 * time.Minute,
	}

	for _, acc := range accounts {
		if acc.Enabled {
			pool.accounts = append(pool.accounts, &Account{
				ID:     acc.ID,
				APIKey: acc.APIKey,
				Weight: acc.Weight,
				Status: AccountStatusHealthy,
			})
		}
	}

	return pool
}

func (p *AccountPool) Select() *Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.accounts) == 0 {
		return nil
	}

	healthyAccounts := make([]*Account, 0)
	for _, acc := range p.accounts {
		if acc.Status == AccountStatusHealthy {
			healthyAccounts = append(healthyAccounts, acc)
		} else if time.Since(acc.LastUsed) > p.recoverTime && acc.ErrorCount >= p.maxErrors {
			healthyAccounts = append(healthyAccounts, acc)
		}
	}

	if len(healthyAccounts) == 0 {
		return nil
	}

	totalWeight := 0
	for _, acc := range healthyAccounts {
		totalWeight += acc.Weight
	}

	if totalWeight == 0 {
		idx := atomic.AddUint64(&p.index, 1) % uint64(len(healthyAccounts))
		return healthyAccounts[idx]
	}

	target := int(atomic.AddUint64(&p.index, 1) % uint64(totalWeight))
	current := 0
	for _, acc := range healthyAccounts {
		current += acc.Weight
		if current > target {
			return acc
		}
	}

	return healthyAccounts[0]
}

func (p *AccountPool) MarkSuccess(accountID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, acc := range p.accounts {
		if acc.ID == accountID {
			acc.Status = AccountStatusHealthy
			acc.ErrorCount = 0
			acc.LastUsed = time.Now()
			break
		}
	}
}

func (p *AccountPool) MarkFailed(accountID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, acc := range p.accounts {
		if acc.ID == accountID {
			acc.ErrorCount++
			acc.LastUsed = time.Now()
			if acc.ErrorCount >= p.maxErrors {
				acc.Status = AccountStatusUnhealthy
			}
			break
		}
	}
}

func (p *AccountPool) GetAccount(accountID string) *Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, acc := range p.accounts {
		if acc.ID == accountID {
			return acc
		}
	}
	return nil
}

func (p *AccountPool) GetAllAccounts() []*Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Account, len(p.accounts))
	copy(result, p.accounts)
	return result
}

func (p *AccountPool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, acc := range p.accounts {
		if acc.Status == AccountStatusHealthy {
			count++
		}
	}
	return count
}
