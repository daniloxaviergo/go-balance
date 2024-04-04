# go-balance

```sh
docker build . -t go-balance --network=host

docker run -it --network=host -p 7070:7070 --rm --name go-balance go-balance

# dev
docker run --network=host -p 7070:7070 -v $(pwd):/app -it --rm --name go-balance go-balance bash
```

O objetivo é escrever um microserviço que tenha uma performance igual ou melhor ao processo utilizando o redlock.

Iniciamos com um serviço http usando o go-gin.

```go
router.GET("/balance", func(c *gin.Context) {
  action     := c.Query("action")
  account_id := c.Query("account_id")
  amount, _  := strconv.ParseFloat(c.Query("amount"), 64)

  mutex.Lock()
  if(action == "plus_funds") {
    plus_funds(balance, account_id, amount)
  } else {
    sub_funds(balance, account_id, amount)
  }
  count[account_id] = count[account_id] + 1
  mutex.Unlock()

  c.String(200, "ok")
})

router.GET("/balancemx2", func(c *gin.Context) {
  action     := c.Query("action")
  account_id := c.Query("account_id")
  amount, _  := strconv.ParseFloat(c.Query("amount"), 64)

  stringKeyLock.Lock(account_id)

  _, ac_exist := accounts_mxx[account_id]
  if (ac_exist == false) {
    accounts_mxx[account_id] = &BalanceMx{ mu: &sync.Mutex{}, balance: 0.0, count: 0}
  }

  if(action == "plus_funds") {
    accounts_mxx[account_id].balance = accounts_mxx[account_id].balance + amount
  } else {
    accounts_mxx[account_id].balance = accounts_mxx[account_id].balance - amount
  }

  accounts_mxx[account_id].count = accounts_mxx[account_id].count + 1
  stringKeyLock.Unlock(account_id)

  c.String(200, "ok")
})
```

No endpoint `/balance`, foi utilizado uma mutex para controlar o acesso a variável `balance`.
Somente uma goroutine tem acesso a variável.

No endpoint `/balancemx2`, foi utilizado uma mutex por account_id.
Somente uma goroutine por account_id poderá fazer alterações no balance.
Caso chege duas requisições para accounts diferentes será processado paralelamente.

```
/balance?action=plus_funds&amount=1&account_id=1
/balance?action=plus_funds&amount=1&account_id=2
```

O script para medir a performance em python:

```py
#!/usr/bin/python

import http.client
import random
import sys
import pdb

from timeit import default_timer as timer
from multiprocessing import Process, Queue, Lock

def c_plus_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancech?action=plus_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def c_sub_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancech?action=sub_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def m_plus_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancemx?action=plus_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def m_sub_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancemx?action=sub_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def m2_plus_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancemx2?action=plus_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def m2_sub_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balancemx2?action=sub_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def plus_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balance?action=plus_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def sub_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/balance?action=sub_funds&amount=1&account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def rails_plus_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.6", port="3000")
  connection.request("GET", f"/plus_funds?account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def rails_sub_funds(account_id):
  connection = http.client.HTTPConnection("192.168.0.6", port="3000")
  connection.request("GET", f"/sub_funds?account_id={account_id}")
  response = connection.getresponse()
  connection.close()
  # print(response.status)

def account_balance(account_id):
  connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  connection.request("GET", f"/account_balance?account_id={account_id}")
  response = connection.getresponse()
  print(response.read().decode())
  connection.close()

def worker(i, queue):
  start = timer()
  # connection = http.client.HTTPConnection("192.168.0.8", port="7070")
  for x in range(100):
    account_id = random.randint(0, 2)

    for account_id in range(3):
      plus_funds(account_id)
      sub_funds(account_id)
    #   m_plus_funds(account_id)
    #   m_sub_funds(account_id)
    #   m2_plus_funds(account_id)
    #   m2_sub_funds(account_id)
    #   rails_plus_funds(account_id)
    #   rails_sub_funds(account_id)
    #   c_plus_funds(account_id)
    #   c_sub_funds(account_id)

    # m_plus_funds(account_id)
    # for account_id in range(3):
    #   m_plus_funds(account_id)
    #   # m_sub_funds(account_id)

  for account_id in range(3):
    account_balance(account_id)

  end = timer() - start
  print(f"{i} {end}")
  queue.put(1)

queues = []
for i in range(10):
  queue = Queue()
  queues.append(queue)
  p = Process(target=worker, args=(i, queue))
  p.start()

# Wait for all processes to finish
for queue in queues:
  queue.get()

breakpoint()
```

Sempre testamos um loop de 100 interações com um loop de account (0..2) para cada account_id fazendo um plus_funds e um sub_funds.
Tendo 10 processos/threads simultâneas.
```py
proc - 1
  for x in range(100):
    for account_id in range(3):
      plus_funds(account_id)
      sub_funds(account_id)
proc - 2
...
proc - 10
```

Ao final de cada processo pedimos o saldo das contas
```json
// /balance
{"account_id":"1","balance":0,"count":2000,"m2_balance":0,"m2_count":0,"m_balance":0,"m_count":0}
{"account_id":"0","balance":0,"count":2000,"m2_balance":0,"m2_count":0,"m_balance":0,"m_count":0}
{"account_id":"2","balance":0,"count":2000,"m2_balance":0,"m2_count":0,"m_balance":0,"m_count":0}

// /balancemx2
{"account_id":"0","balance":0,"count":0,"m2_balance":0,"m2_count":2000,"m_balance":0,"m_count":0}
{"account_id":"1","balance":0,"count":0,"m2_balance":0,"m2_count":2000,"m_balance":0,"m_count":0}
{"account_id":"2","balance":0,"count":0,"m2_balance":0,"m2_count":2000,"m_balance":0,"m_count":0}
```

Na comparação entre os dois casos `/balance` e `/balancemx2`
| proc     | /balance | /balancemx2 | /balance > /balancemx2 |
| -------- | -------- | -------- | -------- |
| 0 | 0.13512477799667977 | 0.13170973699743627 | true |
| 1 | 0.13496244099951582 | 0.13277721800113795 | true |
| 2 | 0.13441712700296193 | 0.13257765800517518 | true |
| 3 | 0.13385261299845297 | 0.13198651200218592 | true |
| 4 | 0.13364401100261603 | 0.13287151699478272 | true |
| 6 | 0.1328332449993468 |  0.13224297899432713 | true |
| 5 | 0.13257571199937956 | 0.1313633000027039 | true |
| 7 | 0.1332333579994156 |  0.1310202719978406 | true |
| 8 | 0.13328375600394793 | 0.1314919779979391 | true |
| 9 | 0.13294003200280713 | 0.13293065200559795 | true |

Executando o teste no ruby `docker container exec -it blue-whale-exchange-1 bundle exec rails runner macros/runner.rb`

```rb
require 'net/http'

class GoGin
  def self.plus_funds(account_id)
    # started_at = (Time.now.to_f * 1_000).to_i
    url = URI("http://192.168.0.8:7070/balance?action=plus_funds&amount=1&account_id=#{account_id}")
    Net::HTTP.get(url)
    # ended_at = "%.3f" % (Time.now.to_f - (started_at / 1000))
    # Rails.logger.info(ended_at)
  end

  def self.sub_funds(account_id)
    url = URI("http://192.168.0.8:7070/balance?action=sub_funds&amount=1&account_id=#{account_id}")
    Net::HTTP.get(url)
  end

  def self.m2_plus_funds(account_id)
    # started_at = (Time.now.to_f * 1_000).to_i
    url = URI("http://192.168.0.8:7070/balancemx2?action=plus_funds&amount=1&account_id=#{account_id}")
    Net::HTTP.get(url)
    # ended_at = "%.3f" % (Time.now.to_f - (started_at / 1000))
    # Rails.logger.info(ended_at)
  end

  def self.m2_sub_funds(account_id)
    url = URI("http://192.168.0.8:7070/balancemx2?action=sub_funds&amount=1&account_id=#{account_id}")
    Net::HTTP.get(url)
  end

  def self.account_balance(account_id)
    url = URI("http://192.168.0.8:7070/account_balance?account_id=#{account_id}")
    Rails.logger.info(Net::HTTP.get(url))
  end
end

def test_gogin(i)
  100.times do |j|
    3.times do |account_id|
      GoGin.plus_funds(account_id)
      GoGin.sub_funds(account_id)
    end
  end
end

started_at = (Time.now.to_f * 1_000).to_i
redlock = RedisRedLock.new
threads = []
10.times do |i|
  threads << Thread.new { test_gogin(i) }
end

threads.map(&:join)
ended_at = "%.3f" % (Time.now.to_f - (started_at / 1000))
Rails.logger.info(ended_at)
```

Comparando o tempo de processamento do python com ruby

| env     | /balance | /balancemx2 |
| -------- | -------- | -------- |
| ruby | 48.068 | 48.513 |
| python | 1.33686 | 1.32097 |



Script utilizado para testar o redlock
```rb
class RedisRedLock
  REDIS_INSTANCE = Redis.new(url: ENV["REDIS_URL_ACCOUNT_BALANCE_VIRTUALIZED"], db: ENV["VIRTUALIZED_BALANCE_DB"])
  LOCK_MANAGER = Redlock::Client.new([REDIS_INSTANCE])

  def plus_funds(account_id)
    begin
      LOCK_MANAGER.lock!("cb_#{account_id}", 8000) do
        balance = REDIS_INSTANCE.get("cb:balance:#{account_id}").to_i
        balance = balance + 1
        REDIS_INSTANCE.set("cb:balance:#{account_id}", balance)
      end
    rescue StandardError => e
      Rails.logger.info "retry plus_funds"
      retry
    end
  end

  def sub_funds(account_id)
    begin
      LOCK_MANAGER.lock!("cb_#{account_id}", 8000) do
        balance = REDIS_INSTANCE.get("cb:balance:#{account_id}").to_i
        balance = balance - 1
        REDIS_INSTANCE.set("cb:balance:#{account_id}", balance)
      end
    rescue StandardError => e
      Rails.logger.info "retry sub_funds"
      retry
    end
  end
end

def test_redlock(redlock, i)
  100.times do |j|
    3.times do |account_id|
      redlock.plus_funds(account_id)
      redlock.sub_funds(account_id)
    end
  end
end
```

Script utilizado para testar o go-balance
```rb
require 'socket'

class GoBalance
  SERVER_HOST = '192.168.0.6'
  SERVER_PORT = 7070

  def initialize
    begin
      # Connect to the server
      @client = TCPSocket.new(SERVER_HOST, SERVER_PORT)
      puts "Connected to #{SERVER_HOST}:#{SERVER_PORT}"
    rescue StandardError => e
      puts "Error: #{e.message}"
    end
  end

  def plus_funds(account_id, amount)
    command = "PLUS_FUNDS #{account_id} #{amount}"

    @client.puts(command)
    response = @client.gets
  end

  def sub_funds(account_id, amount)
    command = "SUB_FUNDS #{account_id} #{amount}"

    @client.puts(command)
    response = @client.gets
  end

  def balance(account_id)
    command = "BALANCE #{account_id}"

    @client.puts(command)
    response = @client.gets
  end

  def close
    @client.close
  end
end

def test_gobalance(i)
  gobalance = GoBalance.new
  100.times do |j|
    3.times do |account_id|
      gobalance.plus_funds(account_id, 1)
      gobalance.sub_funds(account_id, 1)
    end
  end
  gobalance.close
end
```

Comparando o redlock e go-balance

| redlock | redis | go-balance |
| -------- | -------- | -------- |
| 3.976 | 1.345 | 1.05 | 0.494 |

```rb
irb(main):001:1* 3.times do |account_id|
irb(main):002:1*   puts Go.exec("balance #{account_id}")
irb(main):003:0> end
0.000000 2000
0.000000 2000                                                               
0.000000 2000 
```
