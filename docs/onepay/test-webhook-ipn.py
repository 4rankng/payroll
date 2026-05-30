"""
OnePay IPN Webhook Tester

Simulates OnePay sending a signed IPN callback to the webhook endpoint.
Uses the same OWS1-HMAC-SHA256 signing as the official OnePay SDK.

Usage:
    # Test against local server
    python3 test-webhook-ipn.py --target http://localhost:8080

    # Test against prod
    python3 test-webhook-ipn.py --target https://tingting.vip

    # Custom funds_transfer_id (must match an existing wallet_payment)
    python3 test-webhook-ipn.py --target https://tingting.vip --transfer-id FT-12345

    # Test failure state
    python3 test-webhook-ipn.py --target https://tingting.vip --state failed

Requirements:
    pip install requests
"""

import hashlib
import hmac
import json
import time
import argparse
import sys

# ---------------------------------------------------------------------------
# Credentials — UPDATE THESE FOR YOUR ENVIRONMENT
# ---------------------------------------------------------------------------
# Sandbox (default)
PARTNER_ID = "TESTVFICPO"
PARTNER_KEY = "C2B5DA903DAE19E215454211C66A59DD"
ACCOUNT_ID = "666894931888"

# Prod — uncomment and fill when testing against production
# PARTNER_ID = ""
# PARTNER_KEY = ""
# ACCOUNT_ID = ""

# ---------------------------------------------------------------------------
# Signature constants
# ---------------------------------------------------------------------------
SCHEME = "OWS1"
ALGORITHM = "OWS1-HMAC-SHA256"
TERMINATOR = "ows1_request"
OWS_REGION = "onepay"
OWS_SERVICE = "onepayout"
DATE_LAYOUT = "%Y%m%dT%H%M%SZ"


def hmac_sha256(key: bytes, msg: str) -> bytes:
    return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).digest()


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def uri_encode(data: str, encode_slash: bool = False) -> str:
    """URI-encode matching Go's uriEncode: unreserved chars + '/' when encode_slash=False."""
    result = []
    for ch in data.encode("utf-8"):
        if (ord('A') <= ch <= ord('Z') or ord('a') <= ch <= ord('z') or
                ord('0') <= ch <= ord('9') or ch in (ord('_'), ord('-'), ord('~'), ord('.')) or
                (ch == ord('/') and not encode_slash)):
            result.append(chr(ch))
        else:
            result.append(f"%{ch:02X}")
    return "".join(result)


def sign_webhook(
    partner_id: str,
    partner_key: str,
    method: str,
    full_url: str,
    headers: dict,
    signed_header_names: list,
    body: bytes,
    timestamp: time.struct_time,
) -> str:
    """Sign a webhook request using OWS1-HMAC-SHA256."""
    # 1. Canonical URI — OnePay signs IPNs using their internal gateway URL,
    #    not the public IPN URL they PUT to. This must match the hardcoded
    #    onepayIPNInternalURL in signing.go VerifyIPN.
    onepay_internal_url = "http://localhost/payout-merchants/https/tingting.vip/443/api/v1/webhooks/disbursement/1pay"
    canonical_uri = uri_encode(onepay_internal_url)

    # 2. Canonical Headers
    canonical_headers = ""
    sorted_names = sorted(signed_header_names, key=lambda k: k.lower())
    signed_headers_str = ";".join(sorted_names)
    # Build a lowercase-keyed lookup for case-insensitive header matching
    headers_lower = {k.lower(): v for k, v in headers.items()}
    for name in sorted_names:
        canonical_headers += name.lower() + ":" + headers_lower[name.lower()].strip() + "\n"

    # 3. Hashed payload
    hashed_payload = sha256_hex(body) if body else sha256_hex(b"")

    # 4. Canonical Request
    canonical_request = "\n".join([
        method.upper(),
        canonical_uri,
        "",  # no query string
        canonical_headers,
        signed_headers_str,
        hashed_payload,
    ])

    # 5. String to sign
    ts_iso = time.strftime(DATE_LAYOUT, timestamp)
    ts_date = time.strftime("%Y%m%d", timestamp)
    scope = f"{ts_date}/{OWS_REGION}/{OWS_SERVICE}/{TERMINATOR}"
    string_to_sign = "\n".join([
        ALGORITHM,
        ts_iso,
        scope,
        sha256_hex(canonical_request.encode("utf-8")),
    ])

    # 6. Derive signing key
    date_key = hmac_sha256(f"{SCHEME}{partner_key}".encode(), ts_date)
    region_key = hmac_sha256(date_key, OWS_REGION)
    service_key = hmac_sha256(region_key, OWS_SERVICE)
    signing_key = hmac_sha256(service_key, TERMINATOR)

    # 7. Compute signature
    signature = hmac_sha256(signing_key, string_to_sign).hex()
    credential = f"{partner_id}/{scope}"

    return (
        f"{ALGORITHM} Credential={credential},"
        f"SignedHeaders={signed_headers_str},"
        f"Signature={signature}"
    )


def send_ipn(
    target: str,
    state: str,
    funds_transfer_id: str,
    partner_id: str,
    partner_key: str,
    account_id: str,
    verbose: bool = False,
):
    """Build and send a signed IPN webhook to the target server."""
    import requests

    path = "/api/v1/webhooks/disbursement/1pay"
    url = target.rstrip("/") + path

    ts = time.gmtime()
    ts_iso = time.strftime(DATE_LAYOUT, ts)

    # Build IPN payload
    ipn_body = {
        "transaction_id": f"TXN-IPN-TEST-{int(time.time())}",
        "funds_transfer_id": funds_transfer_id,
        "funds_transfer_info": "Test IPN from script",
        "account_id": account_id,
        "remark": "webhook test",
        "account_number": "103000614434",
        "holder_name": "NGUYEN VAN A",
        "amount": 50000,
        "currency": "VND",
        "state": state,
        "response_code": "0" if state == "approved" else "101",
        "message": "Success" if state == "approved" else "Failed",
        "swift_code": "ICBVVNVX",
        "create_time": ts_iso,
        "update_time": ts_iso,
        "batch_id": "",
    }
    body_bytes = json.dumps(ipn_body).encode("utf-8")

    # Headers to sign
    signed_header_names = ["accept", "content-type", "x-op-date", "x-op-expires"]
    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json",
        "X-OP-Date": ts_iso,
        "X-OP-Expires": "6000",
    }

    # Sign
    auth = sign_webhook(
        partner_id=partner_id,
        partner_key=partner_key,
        method="PUT",
        full_url=url,
        headers=headers,
        signed_header_names=signed_header_names,
        body=body_bytes,
        timestamp=ts,
    )
    headers["X-OP-Authorization"] = auth

    if verbose:
        print("=" * 60)
        print("IPN PAYLOAD:")
        print(json.dumps(ipn_body, indent=2))
        print("=" * 60)
        print(f"PUT {url}")
        for k, v in headers.items():
            print(f"  {k}: {v}")
        print("=" * 60)

    # Send
    resp = requests.put(url, data=body_bytes, headers=headers, timeout=15)

    print(f"Status: {resp.status_code}")
    try:
        print(f"Body:   {json.dumps(resp.json(), indent=2)}")
    except ValueError:
        print(f"Body:   {resp.text}")

    return resp


def main():
    parser = argparse.ArgumentParser(description="Test OnePay IPN webhook")
    parser.add_argument(
        "--target", default="http://localhost:8080",
        help="Target server base URL (default: http://localhost:8080)"
    )
    parser.add_argument(
        "--state", default="approved",
        choices=["approved", "failed", "reverted", "pending", "created"],
        help="IPN state to send (default: approved)"
    )
    parser.add_argument(
        "--transfer-id", default="FT-TEST-001",
        help="funds_transfer_id to use (must match existing wallet_payment for full processing)"
    )
    parser.add_argument(
        "--partner-id", default=PARTNER_ID,
        help="OnePay partner ID"
    )
    parser.add_argument(
        "--partner-key", default=PARTNER_KEY,
        help="OnePay partner key (HMAC secret)"
    )
    parser.add_argument(
        "--account-id", default=ACCOUNT_ID,
        help="OnePay account ID"
    )
    parser.add_argument(
        "-v", "--verbose", action="store_true",
        help="Print full request details"
    )
    args = parser.parse_args()

    print(f"Sending {args.state} IPN to {args.target}...")
    print(f"  funds_transfer_id: {args.transfer_id}")
    print()

    resp = send_ipn(
        target=args.target,
        state=args.state,
        funds_transfer_id=args.transfer_id,
        partner_id=args.partner_id,
        partner_key=args.partner_key,
        account_id=args.account_id,
        verbose=args.verbose,
    )

    if resp.status_code == 200:
        print("\nWebhook ACCEPTED by server.")
    elif resp.status_code == 400:
        print("\nWebhook REJECTED by server (signature verification failed?).")
        print("Check: partner_key matches the server's ONEPAY_PARTNER_KEY env var.")
    elif resp.status_code == 404:
        print("\nWebhook endpoint NOT FOUND (route not registered?).")
        print("Check: ENABLE_ONEPAY=true in server env.")
    else:
        print(f"\nUnexpected status code: {resp.status_code}")

    sys.exit(0 if resp.status_code == 200 else 1)


if __name__ == "__main__":
    main()
