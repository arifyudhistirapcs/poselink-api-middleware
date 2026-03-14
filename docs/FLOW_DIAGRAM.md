# Flow Diagram - Payment Middleware

## 1. Flow Mapping MID/TID ke Serial Number

```
┌─────────────────────────────────────────────────────────────────┐
│                    SERVER STARTUP                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Load Config      │
                    │ from Env Var     │
                    └──────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │ MIDTID_MAPPINGS (JSON)        │
              │ {                             │
              │   "M001:T001": "SN12345",     │
              │   "M002:T002": "SN67890"      │
              │ }                             │
              └───────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Create           │
                    │ InMemoryMapper   │
                    │ (map in RAM)     │
                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Server Ready     │
                    │ Listening :8080  │
                    └──────────────────┘
```

## 2. Flow Transaction Request

```
┌─────────┐                                    ┌──────────────┐
│   POS   │                                    │  Middleware  │
└─────────┘                                    └──────────────┘
     │                                                 │
     │  POST /api/v1/transaction                      │
     │  {                                              │
     │    "token": "xxx",                              │
     │    "mid": "M001",                               │
     │    "tid": "T001",                               │
     │    "trx_id": "TRX-001"                          │
     │  }                                              │
     ├────────────────────────────────────────────────>│
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 1. Validate Request │
     │                                    │    - Check trx_id   │
     │                                    └─────────────────────┘
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 2. Lookup Mapping   │
     │                                    │    MID:M001         │
     │                                    │    TID:T001         │
     │                                    │    ↓                │
     │                                    │    Key: "M001:T001" │
     │                                    └─────────────────────┘
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 3. Get Serial       │
     │                                    │    mappings.get()   │
     │                                    │    → "SN12345"      │
     │                                    └─────────────────────┘
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 4. Store Tx         │
     │                                    │    Status: PENDING  │
     │                                    └─────────────────────┘
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 5. Publish to Ably  │
     │                                    │    Channel:         │
     │                                    │    "edc:SN12345"    │
     │                                    │    Event:           │
     │                                    │    "payment_request"│
     │                                    │    Data: token      │
     │                                    └─────────────────────┘
     │                                                 │
     │                                                 ▼
     │                                    ┌─────────────────────┐
     │                                    │ 6. Wait for         │
     │                                    │    Response         │
     │                                    │    (60 seconds)     │
     │                                    └─────────────────────┘
     │                                                 │
     │                                    ┌────────────┴────────────┐
     │                                    │                         │
     │                                    ▼                         ▼
     │                        ┌──────────────────┐    ┌──────────────────┐
     │                        │ Response Received│    │ Timeout (60s)    │
     │                        │ Status: SUCCESS  │    │ Status: TIMEOUT  │
     │                        └──────────────────┘    └──────────────────┘
     │                                    │                         │
     │<───────────────────────────────────┴─────────────────────────┘
     │  Return Response
     │
```

## 3. Flow Lookup Detail

```
┌──────────────────────────────────────────────────────────────┐
│                    LOOKUP PROCESS                             │
└──────────────────────────────────────────────────────────────┘

Input: MID = "M001", TID = "T001"
         │
         ▼
┌─────────────────────┐
│ 1. Create Key       │
│    key = "M001:T001"│
└─────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ 2. Lookup in Memory Map             │
│                                     │
│    mappings = {                     │
│      "M001:T001": "SN12345",  ◄──── Found!
│      "M002:T002": "SN67890"         │
│    }                                │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────┐
│ 3. Return Result    │
│    "SN12345"        │
└─────────────────────┘
         │
         ▼
┌─────────────────────┐
│ 4. Build Channel    │
│    "edc:SN12345"    │
└─────────────────────┘


JIKA TIDAK DITEMUKAN:
         │
         ▼
┌─────────────────────────────────────┐
│ mappings.get("M999:T999")           │
│ → Not Found!                        │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ Return Error:                       │
│ "unknown mid/tid combination:       │
│  M999:T999"                         │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ HTTP Response:                      │
│ Status: 404                         │
│ Body: {                             │
│   "error": "unknown mid/tid         │
│             combination"            │
│ }                                   │
└─────────────────────────────────────┘
```

## 4. Data Structure di Memory

```
┌────────────────────────────────────────────────────────────┐
│                    SERVER MEMORY                            │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │ MIDTIDMapper (InMemoryMapper)                    │     │
│  ├──────────────────────────────────────────────────┤     │
│  │ mappings: map[string]string                      │     │
│  │                                                   │     │
│  │   Key          │  Value                          │     │
│  │   ─────────────┼──────────                       │     │
│  │   "M001:T001"  │  "SN12345"                      │     │
│  │   "M002:T002"  │  "SN67890"                      │     │
│  │   "M003:T003"  │  "SN11111"                      │     │
│  │                                                   │     │
│  └──────────────────────────────────────────────────┘     │
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │ TransactionStore (SyncMapStore)                  │     │
│  ├──────────────────────────────────────────────────┤     │
│  │ data: sync.Map                                   │     │
│  │                                                   │     │
│  │   Key          │  Value (Transaction)            │     │
│  │   ─────────────┼──────────────────────           │     │
│  │   "TRX-001"    │  {Status: PENDING, ...}         │     │
│  │   "TRX-002"    │  {Status: SUCCESS, ...}         │     │
│  │                                                   │     │
│  └──────────────────────────────────────────────────┘     │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

## 5. Comparison: In-Memory vs Database

```
┌─────────────────────────────────────────────────────────────┐
│                    IN-MEMORY MAPPING                         │
│                    (Current Implementation)                  │
└─────────────────────────────────────────────────────────────┘

POS Request → Middleware → Memory Lookup (O(1)) → Ably
                              ↑
                              │ < 1ms
                              │
                        ┌─────────────┐
                        │ map[string] │
                        │   string    │
                        └─────────────┘


┌─────────────────────────────────────────────────────────────┐
│                    DATABASE MAPPING                          │
│                    (Alternative)                             │
└─────────────────────────────────────────────────────────────┘

POS Request → Middleware → Database Query → Ably
                              ↑
                              │ 5-50ms
                              │
                        ┌─────────────┐
                        │ PostgreSQL  │
                        │   or Redis  │
                        └─────────────┘


┌─────────────────────────────────────────────────────────────┐
│                    HYBRID (Best of Both)                     │
└─────────────────────────────────────────────────────────────┘

POS Request → Middleware → Redis Cache → Ably
                              ↑ Hit (< 1ms)
                              │
                              │ Miss
                              ▼
                        ┌─────────────┐
                        │ PostgreSQL  │
                        │ (Fallback)  │
                        └─────────────┘
```

## 6. Update Mapping Flow

### Current (In-Memory):
```
1. Edit .env file
   MIDTID_MAPPINGS='{"M001:T001":"SN12345","M004:T004":"SN99999"}'

2. Restart server
   ./payment-middleware

3. New mapping loaded
   ✓ M004:T004 → SN99999 available
```

### With Database:
```
1. Insert to database
   INSERT INTO mid_tid_mappings VALUES ('M004', 'T004', 'SN99999');

2. No restart needed!
   ✓ Immediately available

3. Optional: Clear cache
   redis-cli DEL midtid:M004:T004
```

## Summary

**Current Implementation:**
- ✅ Mapping disimpan di **memory** (RAM)
- ✅ Dikonfigurasi via **environment variable**
- ✅ Lookup sangat cepat (< 1ms)
- ⚠️ Perlu restart untuk update
- ⚠️ Data hilang jika server restart

**Untuk Production:**
- Pertimbangkan menggunakan **database** atau **Redis**
- Support update dinamis tanpa restart
- Persistent storage
- Scalable untuk ribuan mapping
