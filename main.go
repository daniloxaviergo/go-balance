package main

import (
  "bufio"
  "fmt"
  "log"
  "net"
  "strings"
  "sync"
  "strconv"
  "github.com/redis/go-redis/v9"
  "context"
)

type Account struct {
  balance float64
  count int
}

type AccountKeyLock struct {
  locks map[string]*sync.Mutex

  mapLock sync.Mutex
}

func NewAccountKeyLock() *AccountKeyLock {
  return &AccountKeyLock{locks: make(map[string]*sync.Mutex)}
}

func (l *AccountKeyLock) getLockBy(key string) *sync.Mutex {
  l.mapLock.Lock()
  defer l.mapLock.Unlock()

  ret, found := l.locks[key]
  if found {
      return ret
  }

  ret = &sync.Mutex{}
  l.locks[key] = ret
  return ret
}

func (l *AccountKeyLock) Lock(key string) {
  l.getLockBy(key).Lock()
}

func (l *AccountKeyLock) Unlock(key string) {
  l.getLockBy(key).Unlock()
}

// func (l *AccountKeyLock) PlusFunds(key string, amount string) float64 {
//   l.getLockBy(key).Lock()
//   defer l.getLockBy(key).Unlock()
//   f_amount, _ := strconv.ParseFloat(amount, 64)

//   l.balance = l.balance + f_amount
//   return l.balance
// }

// func (l *AccountKeyLock) SubFunds(key string, amount string) float64 {
//   l.getLockBy(key).Lock()
//   defer l.getLockBy(key).Unlock()
//   f_amount, _ := strconv.ParseFloat(amount, 64)

//   l.balance = l.balance - f_amount
//   return l.balance
// }

// func (l *AccountKeyLock) GetBalance(key string) float64 {
//   l.getLockBy(key).Lock()
//   defer l.getLockBy(key).Unlock()
//   return l.balance
// }

func save_redis(account_id string, balance float64, count int) {
  client := redis.NewClient(&redis.Options{
    Addr:   "192.168.0.6:6379",
    Password: "", // no password set
    DB:     1,  // use default DB
  })

  ctx := context.Background()
  key_balance := "go:balance:" + account_id
  val_balance := fmt.Sprintf("%f", balance)
  client.Set(ctx, key_balance, val_balance, 0).Err()

  key_count := "go:count:" + account_id
  val_count := fmt.Sprintf("%d", count)
  client.Set(ctx, key_count, val_count, 0).Err()
}

func change_balance(action string, account_id string, amount string, accountKeyLock *AccountKeyLock, accounts map[string]*Account) float64 {
  f_amount, _ := strconv.ParseFloat(amount, 64)
  accountKeyLock.Lock(account_id)

  _, ac_exist := accounts[account_id]
  if (ac_exist == false) {
    accounts[account_id] = &Account{ balance: 0.0, count: 0}
  }

  if(action == "plus_funds") {
    accounts[account_id].balance = accounts[account_id].balance + f_amount
  } else {
    accounts[account_id].balance = accounts[account_id].balance - f_amount
  }

  accounts[account_id].count = accounts[account_id].count + 1
  accountKeyLock.Unlock(account_id)

  // save_redis(account_id, accounts[account_id].balance, accounts[account_id].count)

  return accounts[account_id].balance
}

func main() {
  var accountKeyLock = NewAccountKeyLock()
  accounts := make(map[string]*Account)

  listener, err := net.Listen("tcp", ":7070")
  if err != nil {
    log.Fatalf("Failed to start server: %v", err)
  }
  defer listener.Close()
  fmt.Println("Server listening on port 7070...")

  for {
    conn, err := listener.Accept()
    if err != nil {
      log.Printf("Error accepting connection: %v", err)
      continue
    }

    go handleConnection(conn, accountKeyLock, accounts)
  }
}

func handleConnection(conn net.Conn, accountKeyLock *AccountKeyLock, accounts map[string]*Account) {
  defer conn.Close()
  fmt.Println("Client connected:", conn.RemoteAddr())

  scanner := bufio.NewScanner(conn)
  for scanner.Scan() {
    request := strings.Fields(scanner.Text())
    if len(request) == 0 {
      continue
    }

    switch strings.ToUpper(request[0]) {
    case "PLUS_FUNDS":
      if len(request) != 3 {
        conn.Write([]byte("Invalid PLUS_FUNDS command.\n"))
        continue
      }

      account_id := request[1]
      amount     := request[2]
      // value := accountKeyLock.PlusFunds(account_id, amount)
      fmt.Printf("plus_funds %s", account_id)
      value := change_balance("plus_funds", account_id, amount, accountKeyLock, accounts)

      if value < 0 {
        conn.Write([]byte(fmt.Sprintf("account_id '%s' not found\n", account_id)))
        continue
      }

      fmt.Println(" ok")
      conn.Write([]byte(fmt.Sprintf("%f\n", value)))
    case "SUB_FUNDS":
      if len(request) != 3 {
        conn.Write([]byte("Invalid SUB_FUNDS command.\n"))
        continue
      }

      account_id := request[1]
      amount     := request[2]
      // value := accountKeyLock.SubFunds(account_id, amount)
      fmt.Printf("sub_funds %s", account_id)
      value := change_balance("sub_funds", account_id, amount, accountKeyLock, accounts)

      if value < 0 {
        conn.Write([]byte(fmt.Sprintf("account_id '%s' not found\n", account_id)))
        continue
      }

      fmt.Println(" ok")
      conn.Write([]byte(fmt.Sprintf("%f\n", value)))
    case "BALANCE":
      if len(request) != 2 {
        conn.Write([]byte("Invalid BALANCE command.\n"))
        continue
      }

      account_id := request[1]
      accountKeyLock.Lock(account_id)

      balance := accounts[account_id].balance
      count   := accounts[account_id].count

      accountKeyLock.Unlock(account_id)
      if balance < 0 {
        conn.Write([]byte(fmt.Sprintf("account_id '%s' not found\n", account_id)))
        continue
      }

      conn.Write([]byte(fmt.Sprintf("%f %d\n", balance, count)))
    default:
      conn.Write([]byte("Invalid command\n"))
    }
  }

  if err := scanner.Err(); err != nil {
    log.Printf("Error from connection: %v", err)
  }

  fmt.Println("Client disconnected:", conn.RemoteAddr())
}
