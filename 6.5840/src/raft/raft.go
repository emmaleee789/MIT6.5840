package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

const heartbeatTimeout = 150 * time.Millisecond

const (
	STATE_FOLLOWER = iota
	STATE_CANDIDATE
	STATE_LEADER
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a log
type LogEntry struct {
	Term    int
	Index   int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	state       int // 0: follower state; 1: candidate state; 2: leader state
	currentTerm int
	votedFor    int
	cntVote     int
	cntReply    int
	logs        []LogEntry
	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	/* timers */
	lastHeartbeat   time.Time
	lastElection    time.Time
	electionTimeout time.Duration
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Your code here (2A).
	term := rf.currentTerm
	isleader := rf.state == STATE_LEADER
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int // Candidate's term
	CandidateID  int // Candidate requesting the vote
	LastLogIndex int // Index of candidate's last log entry
	LastLogTerm  int // Term of candidate's last log entry
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // Current term of the responding server
	VoteGranted bool // Indicates if the vote is granted

	ReplyServerID int //for debugging
}

type AppendEntriesArgs struct {
	// Your data here (2A, 2B).
	Term         int // Candidate's term
	LeaderID     int // Candidate requesting the vote
	PrevLogIndex int // Index of candidate's last log entry
	PrevLogTerm  int // Term of candidate's last log entry
	Entries      []LogEntry
	LeaderCommit int // leader's commit index
}

type AppendEntriesReply struct {
	// Your data here (2A).
	Term    int  // Current term of the responding server
	Success bool // Indicates if the vote is granted
}

// example RequestVote RPC handler.
// for candidate (receiver)
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	DPrintf("Server %d receives requestVote from server %d\n", rf.me, args.CandidateID)

	reply.Term = rf.currentTerm
	reply.ReplyServerID = rf.me

	if args.CandidateID == rf.me {
		return
	}
	if args.Term < rf.currentTerm {
		DPrintf("Server %d receives RV with term %d, while it's term is %d", rf.me, args.Term, rf.currentTerm)
		DPrintf("Server %d: bye\n", rf.me)
		reply.VoteGranted = false
	}
	if args.Term > rf.currentTerm {
		DPrintf("Server %d: welcome\n", rf.me)
		reply.VoteGranted = true
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateID
		rf.state = STATE_FOLLOWER
		rf.cntVote = 0
		rf._reset_election_timer()
	}
}

// Receiver implementation:
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Success = false
	} else {
		reply.Success = true
		reply.Term = rf.currentTerm
		if args.Entries == nil { // This is heartbeat
			rf._reset_election_timer()
			rf.cntVote = 0
			rf.currentTerm = args.Term
			rf.state = STATE_FOLLOWER // for disconnect->reconnect case
		}
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.

// asks Raft to start the processing to append the command to the replicated log.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := false

	// Your code here (2B).
	// check if this server is leader
	if rf.state != 2 {
		return index, term, isLeader
	}

	//if it is, start the agreement and return immediately.
	log := rf.AppendEntry_handler(command)
	rf.RPC_handler(0)

	return log.Index, log.Term, true

}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

/* helper function */
func (rf *Raft) _getLastLogTerm() int {
	if len(rf.logs) == 0 {
		return 0
	} else {
		return rf.logs[len(rf.logs)-1].Term
	}
}

/* helper function */
func (rf *Raft) _reset_election_timer() {
	electionTimeout := 300 + (rand.Int63() % 300)
	rf.electionTimeout = time.Duration(electionTimeout) * time.Millisecond
	rf.lastElection = time.Now()

}

func (rf *Raft) pastElectionTimeout() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	f := time.Since(rf.lastElection) > rf.electionTimeout
	return f
}

func (rf *Raft) _reset_heartbeat_timer() {
	rf.lastHeartbeat = time.Now()
}

func (rf *Raft) pastHeartbeatTimeout() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return time.Since(rf.lastHeartbeat) > heartbeatTimeout
}

func (rf *Raft) AppendEntry_handler(command interface{}) LogEntry {
	var log LogEntry

	return log
}

// used by leader
func (rf *Raft) LeaderElection_handler() {
	rf.mu.Lock()
	rf.currentTerm += 1
	rf.votedFor = rf.me
	rf.cntVote += 1
	term := rf.currentTerm

	var args *RequestVoteArgs = &RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: len(rf.logs),
		LastLogTerm:  rf._getLastLogTerm()}

	rf.mu.Unlock()

	// this is this server's out channel, while it also concurrently has in channel
	// 请求投票：
	for i := range rf.peers {
		go func(index int) {
			if index == rf.me {
				return
			}
			var reply *RequestVoteReply = &RequestVoteReply{}
			ok := rf.sendRequestVote(index, args, reply)
			if ok {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				if reply.VoteGranted {
					DPrintf("Server %d receives reply from server %d\n", rf.me, reply.ReplyServerID)
					rf.cntVote += 1
					if rf.state != STATE_CANDIDATE || term != rf.currentTerm { // if another server has become the leader
						return
					}
					if rf.cntVote > len(rf.peers)/2 && rf.state == STATE_CANDIDATE && term == rf.currentTerm { //it concurrently has in channel, which may change its state
						DPrintf("Server %d is leader!\n", rf.me)
						rf.state = STATE_LEADER
						rf.cntVote = 0
						// done = true
						rf._reset_heartbeat_timer()
						rf.RPC_handler(1) //选举成功立刻发送
					}
				}
				if reply.Term > rf.currentTerm {
					rf.state = STATE_FOLLOWER
					rf.currentTerm = reply.Term
					rf.votedFor = -1
					rf._reset_election_timer()
					return
				}
			} else {
				DPrintf("Server %d gets no reply from server %d\n", rf.me, index)
			}
		}(i) // bug log
	}
}

// when a server becomes the leader... send empty RPC
// or
// when a server what to append entries... send AppendEntries RPC
func (rf *Raft) RPC_handler(is_heartbeat int) {
	var args *AppendEntriesArgs
	switch is_heartbeat {
	case 1:
		for i := range rf.peers {
			go func(index int) {
				if index == rf.me {
					return
				}
				args = &AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderID:     rf.me,
					PrevLogIndex: 0,
					PrevLogTerm:  0,
					Entries:      nil,
					LeaderCommit: rf.commitIndex}
				var reply *AppendEntriesReply = &AppendEntriesReply{}
				ok := rf.sendAppendEntries(index, args, reply)
				if ok && reply.Success {
					DPrintf("Server %d knows that server %d is the leader in term %d\n", index, rf.me, rf.currentTerm)
				}
			}(i)
		}
		return
	}
	// else {
	// 	args = &AppendEntriesArgs{
	// 		Term:         rf.currentTerm,
	// 		LeaderID:     rf.me,
	// 		PrevLogIndex: rf.nextIndex[index] - 1, // wrong
	// 		PrevLogTerm:  rf.logs[rf.nextIndex[index]-1].Term, // wrong
	// 		Entries:      rf.logs[rf.nextIndex[index]], //not sure // wrong
	// 		LeaderCommit: rf.commitIndex}
	// }
}

func (rf *Raft) ticker() {
	for !rf.killed() {
		switch rf.state {
		case STATE_FOLLOWER:
			if rf.pastElectionTimeout() {
				rf.mu.Lock()
				rf.state = STATE_CANDIDATE
				rf.mu.Unlock()
				rf.LeaderElection_handler()
			}
		case STATE_CANDIDATE:
			if rf.pastElectionTimeout() {
				rf.LeaderElection_handler()
			}
		case STATE_LEADER:
			if rf.pastHeartbeatTimeout() {
				rf._reset_heartbeat_timer()
				rf.RPC_handler(1)
			}
		}
		time.Sleep(time.Duration(40+(rand.Int63()%10)) * time.Millisecond)

	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = STATE_FOLLOWER
	rf.currentTerm = 0
	rf.votedFor = -1 /* server's index start from 0 */
	rf.cntVote = 0
	rf.cntReply = 0
	rf.logs = make([]LogEntry, 1) /* log's index start from 1 */
	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))

	//initialized to leader last log index + 1
	for i := range rf.nextIndex {
		go func(index int) {
			rf.nextIndex[index] = 1
		}(i)
	}

	rf.mu.Lock()
	rf._reset_election_timer()
	rf.mu.Unlock()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
