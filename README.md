# MIT6.5840

## Lab2A

#### 2.1 **func (rf *Raft) ticker():**

##### 首先检查 rf 是否被 killed，如果否，则建立选举机制：

接着，看 rf的 state：

1. Follower：如果 timer time out，则变为 candidate

2. Candidate：发起选举

3. Leader：给其他 server 发送heartbeat（empty AppendEntriesRPC）

**2.1.1 Candidate发起选举流程：**（注意和 RequestVote 函数不同，本函数是发送端，RequestVote函数是接收端）

服务器应将`requestVote`消息发送给所有对等方。消息参数应包含服务器的current term、候选 ID、最后一个日志索引`len(rf.logs)`以及最后一个日志的术语（在实验 2A 中始终为 0）。请注意，这些 RPC 消息应该并行发送，因此我们为每个 RPC 调用启动一个 goroutine，并创建一个通道来接收来自同级的投票结果。

当所有消息发送出去后，它会继续监听结果通道来统计投票结果。如果成功接收到响应，则检查投票是否被授予，否则，认为投票未被授予（或者服务器可以继续发送请求投票，直到收到响应）。

如果收到多数票，服务器就会声称它赢得了选举并广播当局，那么结束选举。但在此之前，有一个极端的情况，即服务器状态更改为follower，因为在计算投票结果时，另一台服务器已经赢得了选举。在这种情况下，服务器除了终止选举之外什么也不做。

或者，如果接受到了其他 leader 的心跳，那么结束选举（或者，如果回复的 term 大于自己的 term，那么结束选举）--> empty RTC&non-empty RTC

**2.1.2 Leader 广播空 RPC 流程：**（注意和 AppendEntries 函数不同，本函数是发送端，RequestVote函数是接收端）

**2.1.3 什么时候重置 election timer：**

当 candidate 发起新一轮选举时

当 candidate 变为 follower 时

当 follower 收到 leader 的empty RPC（心跳）/ RPC时

- 注意需要考虑当 election timer 没有结束（第三种情况）时，需要取消计时器



#### **2.2 func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) ：**

如果请求中的任期较小，则为陈旧选举，应拒绝该请求。

如果请求中的任期高于服务器的任期，则意味着正在发生新的选举，如果服务器曾经对先前的任期进行过投票，则服务器应重置其投票。

否则，如果服务器尚未投票或服务器已投票给该候选人（如果 RPC 响应未到达候选人且重试 RPC），则服务器应对以下两种情况之一进行投票：

- 候选人的最后一个任期高于服务器的最后一个任期。
- 候选者的最后一个任期等于服务器的最后一个任期，但候选者的日志数等于或大于服务器的日志数。
