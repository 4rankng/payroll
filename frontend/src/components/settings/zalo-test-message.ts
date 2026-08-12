const SALARY_SAMPLE_TEMPLATE_ID = "619686";

const formatVietnameseDate = (date: Date) => {
  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  return `${day}/${month}/${date.getFullYear()}`;
};

export const buildSalaryZnsTestPayload = (now = new Date()) => {
  const expiryDate = new Date(now);
  expiryDate.setDate(expiryDate.getDate() + 30);

  return {
    template_id: SALARY_SAMPLE_TEMPLATE_ID,
    template_data: {
      customer_name: "Nhân viên kiểm thử",
      max_amount: "1000000",
      expiry_date: formatVietnameseDate(expiryDate),
    },
  };
};
