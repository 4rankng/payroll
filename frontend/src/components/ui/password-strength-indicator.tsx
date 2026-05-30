import { useState, useEffect } from 'react';
import { validatePassword } from '@/lib/validation';
import { cn } from '@/lib/utils';
import { Check, X } from 'lucide-react';

interface PasswordStrengthIndicatorProps {
  password: string;
  showRequirements?: boolean;
  className?: string;
}

export const PasswordStrengthIndicator: React.FC<PasswordStrengthIndicatorProps> = ({
  password,
  showRequirements = true,
  className
}) => {
  const [validation, setValidation] = useState<ReturnType<typeof validatePassword>>({
    valid: false,
    strength: 'weak',
    errors: []
  });

  useEffect(() => {
    if (password) {
      setValidation(validatePassword(password));
    } else {
      setValidation({
        valid: false,
        strength: 'weak',
        errors: []
      });
    }
  }, [password]);

  const getStrengthColor = () => {
    switch (validation.strength) {
      case 'strong':
        return 'bg-success';
      case 'medium':
        return 'bg-warning';
      default:
        return 'bg-destructive';
    }
  };

  const getStrengthText = () => {
    if (!password) return '';
    switch (validation.strength) {
      case 'strong':
        return 'Mạnh';
      case 'medium':
        return 'Trung bình';
      default:
        return 'Yếu';
    }
  };

  const requirements = [
    { text: 'Ít nhất 8 ký tự', met: password.length >= 8 },
    { text: 'Ít nhất 1 chữ hoa', met: /[A-Z]/.test(password) },
    { text: 'Ít nhất 1 chữ thường', met: /[a-z]/.test(password) },
    { text: 'Ít nhất 1 số', met: /[0-9]/.test(password) },
    { text: 'Ít nhất 1 ký tự đặc biệt', met: /[!@#$%^&*(),.?":{}|<>]/.test(password) }
  ];

  const strengthPercentage = (requirements.filter(r => r.met).length / requirements.length) * 100;

  return (
    <div className={cn('space-y-2', className)}>
      {password && (
        <>
          <div className="space-y-1">
            <div className="flex items-center justify-between typography-body-medium">
              <span className="text-muted-foreground">Độ mạnh mật khẩu</span>
              <span className={cn(
                'font-medium',
                validation.strength === 'strong' && 'text-success',
                validation.strength === 'medium' && 'text-warning',
                validation.strength === 'weak' && 'text-destructive'
              )}>
                {getStrengthText()}
              </span>
            </div>
            <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full transition-all duration-300',
                  getStrengthColor()
                )}
                style={{ width: `${strengthPercentage}%` }}
              />
            </div>
          </div>

          {showRequirements && (
            <div className="space-y-1 pt-2">
              {requirements.map((req, index) => (
                <div key={index} className="flex items-center gap-2 typography-body-medium">
                  {req.met ? (
                    <Check className="h-4 w-4 text-success flex-shrink-0" />
                  ) : (
                    <X className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                  )}
                  <span className={cn(
                    'transition-colors',
                    req.met ? 'text-success' : 'text-muted-foreground'
                  )}>
                    {req.text}
                  </span>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
};