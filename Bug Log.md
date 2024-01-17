## #2A

### Test (2A): initial election

2024.1.13

1. Timer 问题
2. **LeaderElection_handler()** 中，candidate 当选 leader后发送心跳的循环&并发逻辑

2024.1.15

3. **RequestVote()** **和 LeaderElection_handler()** 逻辑问题：server 1 得到投票超过半数了但没当选 leader，~~原因是在LeaderElection_handler()中设置了 vote for itself，导致RequestVote() 中没有进入到正确的逻辑 if 来给予投票~~ <=这样不能确保 server 都投票给自己

   解决方法：LeaderElection_handler()一开始就投票给自己，而不是在RequestVote() 中处理这个逻辑

<img src="/Users/emmaleee/Desktop/截屏2024-01-13 上午11.11.09.png" alt="截屏2024-01-13 上午11.11.09" style="zoom:35%;" />

4. disconnect leader，超时之后剩下的server没选出来新 leader —— [2h]

   1）**RequestVote()** 投票之后rf.state = STATE_ CANDIDATE 设置成 candidate 了

<img src="/Users/emmaleee/Library/Application Support/typora-user-images/截屏2024-01-15 上午11.08.47.png" alt="截屏2024-01-15 上午11.08.47" style="zoom:37%;" />

​	2）**RequestVote()** 中，设置一个拒绝投票的逻辑是 if args.Term < rf.currentTerm || rf.votedFor >= 0，rf.votedFor >= 0代表已经投票给其他的 server 了，但是这样导致的问题是不会接受新一轮的投票，所以改成 if args.Term < rf.currentTerm || rf.state == STATE_FOLLOWER

​	这样还是不对，需要注意：args.Term == rf.currentTerm的话，也不会投票，因为这样有两种可能：1. 你我都是 candidate，而我已经投票给自己了； 2. 你是之前 disconnect 后加进来的，肯定不投给你；3. 我已经投过票了，自己的 term 已经 increase 了。因此，这三种情况下，都不会投票

​	解决方法：**RequestVote()** 中，主逻辑是 args.Term < rf.currentTerm 以及 args.Term > rf.currentTerm

<img src="/Users/emmaleee/Library/Application Support/typora-user-images/截屏2024-01-15 上午11.27.21.png" alt="截屏2024-01-15 上午11.27.21" style="zoom:37%;" />

5. no quorum 即网络内只有一个节点情况下，不应该有 leader

   原因：之前 term 的 rf.cntVote 没有清零

   解决方法：RequestVote() 和 AppendEntries() （heartbeat）中，如果args.Term > rf.currentTerm 则 rf.cntVote = 0 并且 rf.state = STATE_FOLLOWER

<img src="/Users/emmaleee/Library/Application Support/typora-user-images/截屏2024-01-15 下午8.32.42.png" alt="截屏2024-01-15 下午8.32.42" style="zoom:35%;" />

6. 如果只剩下两个节点，并且 term 2 都投给自己，怎么办

论文原文：

> The third possible outcome is that a candidate neither wins nor loses the election: if many followers become candidates at the same time, votes could be split so that no candidate obtains a majority. When this happens, each candidate will time out and start a new election by incre- menting its term and initiating another round of Request- Vote RPCs. However, without extra measures split votes could repeat indefinitely.



2024.1.16

7. “if a quorum arises, it should elect a leader” 问题应该和TestManyElections2A问题一样，都是断联后选举的问题/断联后重新加入

8. readPersist