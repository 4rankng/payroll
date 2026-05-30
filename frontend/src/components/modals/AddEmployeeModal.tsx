import { useSecureModal } from "@/hooks/useSecureModal";
import { AddEmployeeSheet } from "@/components/sheets/AddEmployeeSheet";
import { useNavigate } from "react-router-dom";
import { toast } from "@/components/ui/sonner";

/**
 * Route-based AddEmployee Modal
 * Integrates with the secure modal system for deep-linking
 */
export function AddEmployeeModal() {
  const navigate = useNavigate();
  
  const modal = useSecureModal('add_employee', {
    requiresAuth: true,
    onError: (error) => {
      console.error('AddEmployeeModal error:', error);
      toast({
        title: "Lỗi",
        description: "Có lỗi xảy ra khi tải modal.",
        variant: "destructive"
      });
    }
  });

  const handleClose = () => {
    modal.close();
    navigate(-1); // Go back in history
  };

  return (
    <AddEmployeeSheet
      isOpen={modal.isOpen}
      onClose={handleClose}
    />
  );
}