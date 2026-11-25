# remote-machines

## ADDED Requirements

### Requirement: Machine registry

Cockpit MUST allow the operator to register, list, and remove named remote machines with a host, SSH user, and optional T3 port.

#### Scenario: Add a machine
- **GIVEN** no machine named "oscar"
- **WHEN** `cockpit machine add oscar --host oscar --ssh-user faris --t3-port 3773`
- **THEN** the machine is persisted and appears in `cockpit machine list`

### Requirement: Reachability check

Cockpit MUST aggregate a per-machine health check covering tailnet reachability, SSH auth, T3 service status, and HTTP 200 on the T3 port, without hanging on interactive auth.

#### Scenario: Check reachable machine
- **GIVEN** "oscar" is online and its T3 service is running
- **WHEN** `cockpit machine check oscar`
- **THEN** the result reports all four steps passing and a reachable web URL

#### Scenario: SSH requires approval
- **GIVEN** SSH to the machine prompts for Tailscale web approval
- **WHEN** the check runs with BatchMode
- **THEN** it reports the SSH step failed with an actionable message and does not hang

### Requirement: T3 Code web access

Cockpit MUST expose the T3 Code web URL for a machine and be able to mint a pairing token.

#### Scenario: Open T3 Code
- **GIVEN** a reachable machine with T3 service
- **WHEN** the operator requests its T3 info
- **THEN** cockpit returns the web URL and, on demand, a pairing token + expiry

### Requirement: Adapter integration

The Tailscale and T3 Code adapters MUST implement the existing `adapters.Adapter` interface and be included in `adapters.All()`.

#### Scenario: Adapter status
- **GIVEN** the adapters are registered
- **WHEN** `/api/adapters` is called
- **THEN** Tailscale and T3 entries appear with availability and status payloads