# MIT6.5840

## Lab2B

### Log replication 流程

- 客户端请求

- Leader将命令作为新条目追加到日志中
- Leader并行发送AppendEntries rpc与其他每个服务器，以复制条目。【1’】
- follower收到上述的AppendEntries rpc，执行Consistency Check 【2‘】

> Consistency Check：在其日志中查找具有相同索引和term的条目——leader必须找到两个日志一致的最新日志条目，删除follower日志中在该点之后的所有条目，并将leader在该点之后的所有条目发送给follower。
>
> ​	^------ 具体流程利用nextindex：
>
> ​	The leader maintains a *nextIndex* for each follower, which is the index of the next log entry the leader will send to that follower. When a leader first comes to power, it initializes all nextIndex values to the index just after the last one in its log (11 in Figure 7). If a follower’s log is inconsistent with the leader’s, the AppendEntries consistency check will fail in the next AppendEntries RPC. After a rejection, the leader decrements nextIndex and retries the AppendEntries RPC. Eventually nextIndex will reach a point where the leader and follower logs match. When this happens, AppendEntries will succeed, which removes any conflicting entries in the follower’s log and appends entries from the leader’s log (if any). Once AppendEntries succeeds, the follower’s log is consistent with the leader’s, and it will remain that way for the rest of the term.

- Leader 收到成功消息（？）【3’】

- Leader将该条目“commit”于其状态机，并将执行的结果返回给客户机。【4’】【5’】

> 这也会提交leader日志中所有之前的条目，包括以前的leader创建的条目。

> leader跟踪它知道要提交的最高索引，并将该索引包含在未来的AppendEntries rpc中(包括心跳)

> 一旦追随者了解到日志条目已提交，它将该条目应用于其本地状态机(按日志顺序)。



#### All servers:

【5‘】

- If commitIndex > lastApplied: increment lastApplied, apply log[lastApplied] to state machine (§5.3)

#### Leader:

【1‘】

- If last log index ≥ nextIndex for a follower: send AppendEntries RPC with log entries starting at nextIndex

  【3‘】

  - If successful: update nextIndex and matchIndex for

    follower (§5.3)

  - If AppendEntries fails because of log inconsistency:

    decrement nextIndex and retry (§5.3)

【4‘】

- If there exists an N such that N > commitIndex, a majority

  of matchIndex[i] ≥ N, and log[N].term == currentTerm: set commitIndex = N (§5.3, §5.4).

#### **Receiver implementation:**

【2’】

1. Reply false if term < currentTerm (§5.1)
2. Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
4. Append any new entries not already in the log
5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)



### Q&A

1. 为什么需要 nextIndex：

- 为了consistency check向前匹配时，给leader提供各个 server 的要给他们各自发哪一个 log 的信息

2. 为什么需要commitIndex：

- 为了各个 server 看哪些 log 需要 apply 给状态机

3. 为什么需要leaderCommit：

- 为了指示各个 server 哪些需要 commit，即 commitindex 需要更新到哪里

4. 为什么需要matchindex：

- 为了指示各个 server 是否已经成功被 replicate 了某个 log，即通过最高 log index 来指示

5. 为什么需要prevLogIndex：

- 为了指示最新的相同的log，来consistency check之后，把最后一个相同的 log 位置之后的 logs 全都复制给 follower

6. 为什么需要prevLogTerm：

- 为了指示最新的相同的log，来consistency check之后，把最后一个相同的 log 位置之后的 logs 全都复制给 follower




