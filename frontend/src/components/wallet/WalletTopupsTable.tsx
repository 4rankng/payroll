// DEPRECATED — kept for backward compatibility.
//
// The standalone topups table was replaced by the unified
// `WalletTransactionsList` (see WALLET_FEATURE_PLAN §6.2 / P3 frontend
// slice). Existing callers are forwarded transparently. New code
// should import `WalletTransactionsList` directly.
import WalletTransactionsList from "./WalletTransactionsList";

export default function WalletTopupsTable() {
  return <WalletTransactionsList />;
}
