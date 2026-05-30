"""
OnePay PayOut — Account check + Funds transfer test script.

Based on the official OnePay sample code at:
  docs/onepay/OnePayOWS/Python/

Run:
    python3 test-script.py
"""

import hashlib
import hmac
import time
import json
import urllib.parse

import requests

# ---------------------------------------------------------------------------
# Credentials (MTF sandbox)
# ---------------------------------------------------------------------------
PARTNER_ID  = "TESTVFICPO"
PARTNER_KEY = "C2B5DA903DAE19E215454211C66A59DD"
ACCOUNT_ID  = "666894931888"
BASE_URL    = "https://mtf.onepay.vn"

# Happy-case test recipient from the OnePay docs
TEST_SWIFT_CODE     = "ICBVVNVX"      # VietinBank (from official sample)
TEST_ACCOUNT_NUMBER = "103000614434"  # from official sample
TEST_HOLDER_NAME    = "Nguyen Danh Hoang"
TEST_AMOUNT         = "10000"         # min 10,000 VND

# ---------------------------------------------------------------------------
# Signature constants — matching official Authorization.py
# ---------------------------------------------------------------------------
SCHEME      = "OWS1"
ALGORITHM   = "OWS1-HMAC-SHA256"
TERMINATOR  = "ows1_request"
OWS_REGION  = "onepay"
OWS_SERVICE = "onepayout"
DATE_LAYOUT = "%Y%m%dT%H%M%SZ"

EMPTY_BODY_SHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"


# ---------------------------------------------------------------------------
# Helpers — matching official Util.py + Authorization.py
# ---------------------------------------------------------------------------
def hmac_sha256(key: bytes, msg: str) -> bytes:
    return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).digest()


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def uri_encode(data: str, encode_slash: bool) -> str:
    if encode_slash:
        return urllib.parse.quote(data.encode("utf-8"), safe="~")
    else:
        return urllib.parse.quote(data.encode("utf-8"), safe="~/")


def get_bytes(value: str) -> bytes:
    return value.encode("utf-8")


# ---------------------------------------------------------------------------
# Signing — exactly matching official Authorization.py sign()
# ---------------------------------------------------------------------------
def sign_request(
    access_key_id: str,
    secret_key: str,
    region: str,
    service: str,
    http_method: str,
    uri: str,
    query_parameters: dict,
    signed_headers: dict,
    payload: bytes,
    timestamp: time.struct_time,
) -> str:
    # 1. Canonical URI
    canonical_uri = uri_encode(uri, False)

    # 2. Canonical Query String — sort by raw key, then encode key=value
    canonical_query_string = ""
    for key, value in sorted(query_parameters.items(), key=lambda kv: kv[0]):
        if len(canonical_query_string) > 0:
            canonical_query_string += "&"
        canonical_query_string += uri_encode(str(key), True) + "=" + uri_encode(str(value), True)

    # 3. Canonical Headers + SignedHeaders
    canonical_headers = ""
    signed_header_names = ""
    for key in sorted(signed_headers.keys(), key=lambda k: k.lower()):
        canonical_headers += key.lower() + ":" + str(signed_headers[key]).strip() + "\n"
        if len(signed_header_names) > 0:
            signed_header_names += ";"
        signed_header_names += key.lower()

    # 4. Hashed Payload — matching official logic
    if payload is not None:
        if len(payload) > 0:
            hashed_payload = sha256_hex(payload)
        else:
            hashed_payload = EMPTY_BODY_SHA256
    else:
        hashed_payload = "UNSIGNED-PAYLOAD"

    # 5. Canonical Request
    canonical_request = (
        http_method + "\n"
        + canonical_uri + "\n"
        + canonical_query_string + "\n"
        + canonical_headers + "\n"
        + signed_header_names + "\n"
        + hashed_payload
    )

    # 6. StringToSign
    ts_iso = time.strftime(DATE_LAYOUT, timestamp)
    ts_date = time.strftime("%Y%m%d", timestamp)
    scope = ts_date + "/" + region + "/" + service + "/" + TERMINATOR
    string_to_sign = (
        ALGORITHM + "\n"
        + ts_iso + "\n"
        + scope + "\n"
        + sha256_hex(get_bytes(canonical_request))
    )

    # 7. Signing Key — 4 layers: date → region → service → terminator
    date_key = hmac_sha256(get_bytes(SCHEME + secret_key), ts_date)
    date_region_key = hmac_sha256(date_key, region)
    date_region_service_key = hmac_sha256(date_region_key, service)
    signing_key = hmac_sha256(date_region_service_key, TERMINATOR)

    # 8. Signature
    signature = hmac_sha256(signing_key, string_to_sign).hex()

    credential = access_key_id + "/" + scope

    print("===== CanonicalRequest =====")
    print(repr(canonical_request))
    print("===== StringToSign =====")
    print(repr(string_to_sign))
    print("===== Signing Key =====")
    print(signing_key.hex())
    print("===== Signature =====")
    print(signature)

    return (
        ALGORITHM
        + " Credential=" + credential
        + ",SignedHeaders=" + signed_header_names
        + ",Signature=" + signature
    )


def _send_request(method, url, headers, body=None):
    """Send a request and print the raw HTTP exchange + response."""
    req = requests.Request(method, url, headers=headers, data=body)
    prepared = req.prepare()

    print()
    print("========== RAW HTTP REQUEST ==========")
    print(f"{prepared.method} {prepared.url} HTTP/1.1")
    for k, v in prepared.headers.items():
        print(f"{k}: {v}")
    if body:
        print()
        print(body.decode("utf-8") if isinstance(body, bytes) else body)
    print("=======================================")
    print()

    resp = requests.Session().send(prepared, timeout=15)
    print(f"Status: {resp.status_code}")
    try:
        print(json.dumps(resp.json(), indent=2, ensure_ascii=False))
    except ValueError:
        print(resp.text)
    return resp


# ---------------------------------------------------------------------------
# 1. GET /onepayout/api/v1/customers — Account check
# ---------------------------------------------------------------------------
def test_check_account():
    print("\n" + "=" * 60)
    print("TEST 1: Account Check (GET /customers)")
    print("=" * 60)

    ts = time.gmtime()
    ts_iso = time.strftime(DATE_LAYOUT, ts)
    request_id = f"REQ{int(time.time())}"

    header_sign = {
        "Accept":       "application/json",
        "Content-Type": "application/json",
        "X-OP-Date":    ts_iso,
        "X-OP-Expires": "3600",
    }

    query_param_map = {
        "request_id":     request_id,
        "swift_code":     TEST_SWIFT_CODE,
        "account_number": TEST_ACCOUNT_NUMBER,
        "amount":         TEST_AMOUNT,
        "account_id":     ACCOUNT_ID,
    }

    query_params_encoded = {}
    query_after = "?"
    for key, value in sorted(query_param_map.items(), key=lambda kv: kv[0]):
        query_params_encoded[key] = urllib.parse.quote(str(value), encoding="utf-8")
        query_after += key + "=" + urllib.parse.quote(str(value), encoding="utf-8") + "&"
    query_after = query_after.rstrip("&")

    uri = "/onepayout/api/v1/customers"

    auth_string = sign_request(
        access_key_id=PARTNER_ID,
        secret_key=PARTNER_KEY,
        region=OWS_REGION,
        service=OWS_SERVICE,
        http_method="GET",
        uri=uri,
        query_parameters=query_params_encoded,
        signed_headers=header_sign,
        payload=get_bytes(""),
        timestamp=ts,
    )

    url_request = BASE_URL + uri + query_after
    header_request = {
        "Accept":             "application/json",
        "Content-Type":       "application/json",
        "X-OP-Date":          ts_iso,
        "X-OP-Authorization": auth_string,
        "X-OP-Expires":       "3600",
    }

    return _send_request("GET", url_request, header_request)


# ---------------------------------------------------------------------------
# 2. PUT /onepayout/api/v1/accounts/{id}/funds_transfers/{id} — Transfer
# ---------------------------------------------------------------------------
def test_funds_transfer():
    print("\n" + "=" * 60)
    print("TEST 2: Funds Transfer (PUT /accounts/.../funds_transfers/...)")
    print("=" * 60)

    ts = time.gmtime()
    ts_iso = time.strftime(DATE_LAYOUT, ts)
    funds_transfer_id = f"TF{int(time.time())}"

    uri = f"/onepayout/api/v1/accounts/{ACCOUNT_ID}/funds_transfers/{funds_transfer_id}"

    body = json.dumps({
        "swift_code":         TEST_SWIFT_CODE,
        "account_number":     TEST_ACCOUNT_NUMBER,
        "holder_name":        TEST_HOLDER_NAME,
        "amount":             TEST_AMOUNT,
        "currency":           "VND",
        "funds_transfer_info": funds_transfer_id,
        "remark":             "test transfer from script",
    })
    body_bytes = get_bytes(body)

    header_sign = {
        "Accept":       "application/json",
        "Content-Type": "application/json",
        "X-OP-Date":    ts_iso,
        "X-OP-Expires": "3600",
    }

    auth_string = sign_request(
        access_key_id=PARTNER_ID,
        secret_key=PARTNER_KEY,
        region=OWS_REGION,
        service=OWS_SERVICE,
        http_method="PUT",
        uri=uri,
        query_parameters={},
        signed_headers=header_sign,
        payload=body_bytes,
        timestamp=ts,
    )

    url_request = BASE_URL + uri
    header_request = {
        "Accept":             "application/json",
        "Content-Type":       "application/json",
        "X-OP-Date":          ts_iso,
        "X-OP-Authorization": auth_string,
        "X-OP-Expires":       "3600",
    }

    return _send_request("PUT", url_request, header_request, body=body_bytes)


if __name__ == "__main__":
    resp = test_check_account()
    if resp.status_code == 200:
        data = resp.json()
        if data.get("response_code") == "00":
            print("\n✅ Account check passed, proceeding to transfer...")
            test_funds_transfer()
        else:
            print(f"\n❌ Account check failed: {data.get('response_code')} {data.get('message')}")
    else:
        print(f"\n❌ Account check HTTP error: {resp.status_code}")
