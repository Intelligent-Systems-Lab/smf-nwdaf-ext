# Operations And Troubleshooting

## Audience

- Users operating this branch in integration or test environments
- Maintainers triaging runtime failures

## Configuration Checklist

1. Ensure `serviceNameList` includes `nsmf-event-exposure`.
2. For each target UPF, set `nupfEeApiRoot` under `userplaneInformation.upNodes`.
3. Optionally set:
- `nupfEeNfId`
- `nupfEeReqTimeout`
- `nupfEeMaxRetries`
4. Confirm user-plane links and S-NSSAI/DNN mapping match your topology.

## Runtime Verification

### Nsmf API Reachability

Expected base path:
- `/nsmf-event-exposure/v1/subscriptions`

Quick checks:
- POST a valid single-UE subscription payload.
- Confirm `201 Created` with `Location` header.
- DELETE using returned `subId` and confirm `204`.

### UPF Cascade Verification

On create:
- SMF should call UPF create endpoint using configured `nupfEeApiRoot`.
- Local Nsmf state should include returned UPF `Location`/ID.

On delete:
- SMF should attempt UPF delete before local cleanup.
- Even if UPF delete fails, local state should still be removed.

## Common Failure Patterns

### `404 no active session for supi`

Cause:
- no active SMContext for requested SUPI.

Action:
- verify UE is registered and PDU session is active before subscribe.

### `409 multiple active sessions for supi`

Cause:
- resolver found more than one active SMContext for the same SUPI.

Action:
- narrow selection policy or enforce single active context for this flow.

### `404 ue ip address is not available`

Cause:
- `PDUAddress` not allocated yet in selected SMContext.

Action:
- verify PDU session establishment completed.

### `500 upf event exposure apiRoot is not configured`

Cause:
- selected UPF node lacks `nupfEeApiRoot`.

Action:
- set `nupfEeApiRoot` for the UPF in `smfcfg.yaml`.

### `502 UPF_REQUEST_FAILED`

Cause:
- upstream UPF call failed (transport, 5xx, or invalid upstream response).

Action:
- verify UPF endpoint reachability.
- verify UPF returns `201` + `Location` on create.
- adjust timeout/retry in SMF config.

## Multi-AN Topology Notes

If session establishment reports UPF selection failures under multi-gNB setups:
- verify AN-UPF links are defined correctly.
- verify selected S-NSSAI/DNN exists on reachable UPF.
- ensure this branch build includes multi-AN ingress selection fix.

## URR Threshold Note

This branch only enables URR `Volth` trigger when `volumeThreshold > 0`.
This is intentional and required by this team's extended `UPF-EES` test behavior.
