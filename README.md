Table of Contents
=================

* [Redisgo\_task](#redisgo_task)
  * [长短类型分布式场景介绍](#长短类型分布式场景介绍)
  * [Redisgo\_task实现原理](#redisgo_task实现原理)
    * [SetNx(value\+expire)原子性](#setnxvalueexpire原子性)
    * [ReExpire 粒度控制](#ReExpire-粒度控制)
    * [子协程Done()时间点](#子协程done时间点)
    * [子协程中的Ticker](#子协程中的ticker)
    * [Retry Lock 二进制指数退避机制](#Retry-Lock-二进制指数退避机制)
  * [Redisgo\_task唯一外部依赖](#redisgo_task唯一外部依赖)
  * [Redisgo\_task Lock结构](#redisgo_task-lock结构)
  * [Redisgo\_task架构健壮性设计](#redisgo_task架构健壮性设计)
    * [Redisgo\_task可扩展性](#redisgo_task可扩展性)
    * [Redisgo\_task灵活性](#redisgo_task灵活性)
    * [Redisgo\_task可读性](#redisgo_task可读性)
  * [Redisgo\_task性能指标](#redisgo_task性能指标)
  * [Q&amp;A](#qa)
  * [附：](#附)
    * [用例使用过程](#用例使用过程)
    * [产品设计思路借鉴](#产品设计思路借鉴)

# Redisgo_task
一款基于Goland语言实现的Redis分布式锁产品，支持百万级实例/协程并发，适用于各种常见的分布式场景。

## 长短类型分布式场景介绍
目前业务中分布式锁场景依据任务对象所需的Occupied Time可分为两种：**_短任务类型、长任务类型_**。<br/>
**长任务类型**
> 任务A需要在很长的一段时间占有锁，这个时间未知，直至任务A结束，甚至特殊情况下A的周期为业务的全生命周期，才能释放锁，再供任务A/B/C/D/争抢；<br/>

**短任务类型**
> 任务A在极短的时间内可完成，可在已知的时间阈值内，释放锁，再供任务A/B/C/D/争抢；<br/>

**实质上**<br/>
> 两种类型都是对分布式场景下公共资源的一致性保证；<br/>

**功能上**<br/>
> 长任务类型更倾向于实现某单实例的动态切换，如解决实例单点问题等；<br/>
> 短任务类型更倾向于对时刻高并发的限制，如短时间内的流量控制等<br/>

**Redisgo_task两种任务类型都支持。**

## Redisgo_task实现原理
基于Redis SetNx()方法进行封装，创建子协程监听任务执行状态，任务执行中频次拉取Luck的Expire，当Expire在配置的阈值范围内，持续增加Expire，从而确保Luck在任务进行中不会过期。

### SetNx(value+expire)原子性
lock.Conn.Do("SET", lock.Key, lock.Token, "EX", int(lock.TimeOut), "NX")
### ReExpire 粒度控制
当持续增加任务中所属Lock中的Expire时，设计阈值是为了保证持有的Lock Key Expire始终在可控范围内的同时，更好的便于Expire粒度控制
### 子协程Done()时间点
由doneCh <-chan struct{} Channel阻塞控制，在Unlock()中会出发Close()信号
### 子协程中的Ticker
通过Ticker，持续进行Expire Add操作，可有效避免阻塞及单次Expire Add失败的场景，且有效验证当前锁状态，及时Stop
### Retry Lock 二进制指数退避机制
重试机制<br/>
1、瞬时重试；2、等间隔重试；3、随机重试；4、队列重试<br/><br/>
经过考量，网络抖动下，瞬时、等间隔，可能导致重试成功率低，且DDOS现象；而队列重试在当前产品场景不适用；参考TCP重试机制，故设计采用**二进制指数退避算法作为重试机制**；<br/><br/>
采用TCP超时重试等同的二进制指数退避算法，解决在持续重试获取锁场景下，对服务造成的瞬时负载高及网络抖动导致重试成功率底下的问题。


## Redisgo_task唯一外部依赖
```
dir:config/redisgo_task.toml
[redis]
#expire-锁默认expire「单位s」
#cron 持续增加expire频次
#retries_count 持续获取锁重试次数，重试间隔遵循退避算法，于次数耦合
#monitor_try_all 标识是否永久重试抢锁，true/false
host = "IP:HOST"
key = "Redisgo_Task_Lock_key"
monitor_try_all = true
#retries_count = 4.0
# value = "8292884c-a7a7-0050-9778-e47362a8f578"
# expire = "5s"
# cron = "1s"
```
## Redisgo_task Lock结构
```
type RedisLock struct {
	Host           string
	Expire         time.Duration
	Key            string
	Value          string
	Conn           redis.Conn
	Cron           time.Duration
	DoneExpireChan chan struct{}
	Retry          *Retry
}
```
## Redisgo_task架构健壮性设计
考量到产品架构在实用中的健壮性，针对产品的整体架构设计，对实现过程做出了一下方向的调整：
### Redisgo_task可扩展性
对功能实现过程依赖的参数，及功能函数进行封装成不同程度的Struct、Interface，方便后期功能扩展
### Redisgo_task灵活性
对产品功能涉及到的主要环节阈值拆分，抽象为依赖，支持外部配置化，且作为唯一入口，依据实际场景调整，保证配置的灵活性及功能最细粒度的控制
### Redisgo_task可读性
在产品功能实现过程中，添加完整的日志输出，确保逻辑的清晰可读，降低产品的上手难度

## Redisgo_task性能指标
在数据量、实例数量两个维度验证：在高并发场景下每个实例获得锁的成功率一致；<br/>
实验分为三组，分别为样本一、样本二、样本三如下图:<br/>
![image](https://img-blog.csdnimg.cn/20210514173657877.png?x-oss-process=image/watermark,type_ZmFuZ3poZW5naGVpdGk,shadow_10,text_aHR0cHM6Ly9ibG9nLmNzZG4ubmV0L3FxXzM0NDE3NDA4,size_16,color_FFFFFF,t_70)
> 样本一中数据规模每组在10+，较少，两组成功率相差4.2%，无法体现在双实例下，每个实例成功率一致的目标；<br/>
> 样本二中数据规模每组在3w+，尚可，三组成功率均差在0.2%，已经十分接近目标；<br/>
>样本三中数据规模每组在1w+，尚可，四组成功率均差在0.0025%，可以验证并发场景下，实例强锁成功率均等；<br/>
## Q&A
**1、任务执行时长未知，如何保证任务期间，持续占有锁/？**<br/>
任务占有锁，会启子协程频次监听锁TTL，在可控粒度下，持续保障锁的Expire延长更新。<br/>
**2、主任务结束，如何终止AddExpireTime子协程终止/？**<br/>
关联Goland中协程通信，项目实现中采用Channel/Close()方案实现。<br/>
Of crouse，ctx context/Done方案也可行。<br/>
**3、当并发场景下，会不会出现任务A占有锁的同时，Expire时间到期，锁被任务B占用/？**<br/>
较低的概率会出现这种问题。<br/>
当默认Expire时间内ReNewExpire策略无数次失败，才会导致锁到期自动释放，被其他任务占用。<br/>
**4、如何解决1中，任务A假锁，任务B真锁，A执行结束又将Key删除，破坏任务B的问题/？**<br/>
在Unlock()实现中，做del操作之前会进行Value的校验，匹配时进行del操作，且通过Lua脚本保证原子性。<br/>
**5、AddExpireTime逻辑在短类型场景中，是否没必要存在/？**<br/>
AddExpireTime逻辑是为兼容长类型场景设计，在短类型场景中不影响业务逻辑正常进行。<br/>

## 附：
### 实例启用命令
./redisgo_task task --config config/redisgo_task.toml
### 产品设计思路借鉴
[Consul分布式中Luck()/Unluck()实现原理](https://blog.csdn.net/qq_34417408/article/details/116331540)<br/>
[Redis大型网站高并发场景下分布式锁实现原理](https://blog.csdn.net/qq_34417408/article/details/116799087)<br/>
[重试设计之二进制指数退避机制](https://blog.csdn.net/qq_34417408/article/details/116943097)

感兴趣的各位可以打星交流！！！
