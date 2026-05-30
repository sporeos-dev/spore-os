# Licensing Roadmap & Policy

## Component License Matrix

Spore OS is a distributed system composed of multiple repositories. To balance open-source freedom with commercial viability, we use a hybrid licensing model:

| Repository | License | Primary Use Case |
| :--- | :--- | :--- |
| **Hub** | **AGPL-3.0** | The central coordinator. Requires source disclosure if modified or run as a service. |
| **Client Library** | **Apache 2.0** | Permissive. Safe for proprietary applications. |
| **Core Spokes** | **Apache 2.0** | Permissive. Safe for proprietary worker nodes. |
| **Spec** | **Apache 2.0** | Permissive. Implementations can be proprietary. |
| **Install Scripts** | **Apache 2.0** | Permissive. |

## Critical Boundary: Socket-Based Communication

A key architectural feature of Spore OS is that the **Hub** communicates with **Clients** and **Spokes** exclusively via **network sockets** (IPC over TCP/Unix sockets).

### Why This Matters for Licensing
Under the **AGPL-3.0**, the "copyleft" requirement (forcing you to open-source your code) is triggered if you create a **derivative work** or link code into the same process space.

Because our components communicate **only through sockets**:
1.  **No Derivative Work**: A proprietary application using the **Apache 2.0 Client Library** or running a proprietary **Spoke Node** is considered a **separate work** from the AGPL-licensed Hub.
2.  **Safe Integration**: You can build closed-source, proprietary nodes that interact with the Hub without your proprietary code becoming subject to the AGPL.
3.  **The Boundary**: The copyleft boundary stops at the socket. As long as you do not link the Hub's source code directly into your application (e.g., via static/dynamic linking in the same process), your proprietary code remains under your chosen license.

> **Summary**: If you connect to the Hub via sockets, you are safe to keep your node/client code proprietary. If you embed the Hub code directly into your binary, the AGPL applies.

## Current Status & Future Plan

### Current Status: Mixed Licensing
*   **Hub**: Released under **AGPL-3.0**.
*   **Ecosystem (Client, Spokes, Spec, Install)**: Released under **Apache 2.0**.

### Future Plan: Dual Licensing
The copyright holder intends to introduce a **Commercial Proprietary License** for the **Hub** once the project reaches a stable release.

#### Why Dual Licensing?
This model allows us to:
1.  **Maintain a strong open-source community** under AGPL-3.0.
2.  **Sustain development** by offering a paid option for companies that cannot comply with AGPL (e.g., those who want to run a SaaS version of the Hub without sharing their modifications).

#### What This Means for You
*   **Current Users**: You retain all rights granted by the current licenses for the versions you download today.
*   **Future Users**: When the commercial license launches, you will have a choice:
    *   Continue using the **AGPL version** (free, must share modifications).
    *   Purchase a **Commercial License** (paid, keep modifications private).

We are committed to transparency and will announce the commercial launch well in advance. There will be no "rug pull" retroactively changing the license of existing open-source versions.

## Contact
For questions about this roadmap, licensing boundaries, or early commercial interest: **[ ... coming soon ]**