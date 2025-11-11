┌─────────────────────────────────────────────────────────────────────┐
│                     RELATIONSHIP GRAPH EXAMPLE                       │
│                                                                      │
│  Users (Subjects)          Groups/Teams              Resources       │
│  ================          =============             ==========      │
│                                                                      │
│    user:alice ─────member────> team:engineering                     │
│        │                            │                               │
│        │                            │                               │
│        └──────viewer──────┐         │                               │
│                           │         │                               │
│    user:bob ───member─────┤         │                               │
│        │                  ▼         │                               │
│        │               team:frontend│                               │
│        │                  │         │                               │
│        └────editor────┐   │         │                               │
│                       │   │         │                               │
│    user:charlie       │   │         │                               │
│        │              │   │         │                               │
│        └───owner──┐   │   │         │                               │
│                   ▼   ▼   │         │                               │
│                 folder:project-docs │                               │
│                       │   │         │                               │
│                       │   └─member──┘                               │
│                       │                                             │
│                       │             team:engineering#member         │
│                       │                    │                        │
│                       │                    ▼                        │
│                       └─────────────> document:api-spec             │
│                                          editor                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
