import { Building, CreditCard, User, Phone, Mail, Calendar, FileText } from "lucide-react";
import { format } from 'date-fns';
import type { Lender } from "@/types/api/loan.types";
import { cn } from "@/lib/utils";
import { useBank } from "@/hooks/api/useBanks";

interface LenderCardProps {
  lender: Lender;
  onClick?: () => void;
}

export function LenderCard({ lender, onClick }: LenderCardProps) {
  const hasBankAccountDetails = !!(lender.bank_account_number || lender.bank_account_name);
  const hasBankId = !!lender.bank_id;

  // Load bank details if bank_id exists
  const { data: bank, isLoading: isBankLoading } = useBank(lender.bank_id || 0, {
    enabled: hasBankId && !lender.bank // Only fetch if we have bank_id but not the full bank object
  });

  // Determine if we should show bank details section
  const hasBankDetails = hasBankAccountDetails || hasBankId;

  return (
    <div
      onClick={onClick}
      className={cn(
        "bg-card border rounded-xl p-4 space-y-4 transition-all",
        onClick && "cursor-pointer hover:border-primary/50"
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <h3 className="typography-title-medium font-semibold truncate">
            {lender.name}
          </h3>
          {lender.cccd && (
            <p className="typography-body-small text-muted-foreground font-mono mt-1">
              CCCD: {lender.cccd}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 text-muted-foreground ml-4">
          <Calendar className="w-4 h-4" />
          <span className="typography-body-small">
            {format(new Date(lender.created_at), 'dd/MM/yyyy')}
          </span>
        </div>
      </div>

      {/* Contact Information */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {lender.email && (
          <div className="flex items-center gap-2 min-w-0">
            <Mail className="w-4 h-4 text-muted-foreground shrink-0" />
            <span className="typography-body-small text-muted-foreground truncate">
              {lender.email}
            </span>
          </div>
        )}
        {lender.mobile && (
          <div className="flex items-center gap-2">
            <Phone className="w-4 h-4 text-muted-foreground shrink-0" />
            <span className="typography-body-small text-muted-foreground">
              {lender.mobile}
            </span>
          </div>
        )}
      </div>

      {/* Bank Details */}
      {hasBankDetails ? (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-3 border-t">
          <div className="space-y-1">
            <div className="flex items-center gap-1.5">
              <Building className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="typography-body-small text-muted-foreground">Ngân hàng</span>
            </div>
            <p className="typography-body-small font-medium pl-5">
              {isBankLoading ? 'Đang tải...' : (bank?.branch_name || (lender.bank?.branch_name || '-'))}
            </p>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1.5">
              <CreditCard className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="typography-body-small text-muted-foreground">Số TK</span>
            </div>
            <p className="typography-body-small font-medium font-mono pl-5">
              {lender.bank_account_number || '-'}
            </p>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1.5">
              <User className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="typography-body-small text-muted-foreground">Chủ TK</span>
            </div>
            <p className="typography-body-small font-medium pl-5 truncate">
              {lender.bank_account_name || '-'}
            </p>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-2 p-2 bg-amber-50 border border-amber-200 rounded">
          <FileText className="w-4 h-4 text-amber-600 shrink-0" />
          <span className="typography-body-small text-amber-900">
            Chưa có thông tin ngân hàng
          </span>
        </div>
      )}

      {/* Notes Preview */}
      {lender.notes && (
        <div className="pt-3 border-t">
          <p className="typography-body-small text-muted-foreground line-clamp-2">
            {lender.notes}
          </p>
        </div>
      )}
    </div>
  );
}
