import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Users, Mail, Phone, User } from "lucide-react";
import type { Row } from "@tanstack/react-table";

interface Employee {
  id: number;
  name: string;
  email: string;
  phone: string;
  position: string;
  project: string;
  status: string;
  avatar?: string;
}

interface PartnerEmployeesTableProps {
  employees: Employee[];
}

export const PartnerEmployeesTable = ({ employees }: PartnerEmployeesTableProps) => {
  const getStatusBadge = (status: string) => (
    <Badge className="bg-success text-white font-medium">
      <User className="w-3 h-3 mr-1" />
      {status}
    </Badge>
  );

  const columns = [
    {
      accessorKey: "name",
      header: "Họ và tên",
      cell: ({ row }: { row: Row<Employee> }) => (
        <div className="flex items-center gap-3">
          <UserAvatar
            name={row.getValue("name") as string}
            email={row.original.email}
            src={row.original.avatar}
            size="sm"
          />
          <div>
            <p className="typography-label-large text-foreground">{row.getValue("name")}</p>
            <p className="typography-body-small text-muted-foreground">{row.original.position}</p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "email",
      header: "Email",
      cell: ({ row }: { row: Row<Employee> }) => (
        <div className="flex items-center gap-2">
          <Mail className="h-4 w-4 text-muted-foreground" />
          <span>{row.getValue("email")}</span>
        </div>
      ),
    },
    {
      accessorKey: "phone",
      header: "Điện thoại",
      cell: ({ row }: { row: Row<Employee> }) => (
        <div className="flex items-center gap-2">
          <Phone className="h-4 w-4 text-muted-foreground" />
          <span>{row.getValue("phone")}</span>
        </div>
      ),
    },
    {
      accessorKey: "project",
      header: "Dự án",
      cell: ({ row }: { row: Row<Employee> }) => (
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 bg-primary rounded-full"></div>
          <span className="typography-label-medium text-foreground">{row.getValue("project")}</span>
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      cell: ({ row }: { row: Row<Employee> }) => getStatusBadge(row.getValue("status")),
    },
  ];

  const mobileFields = [
    {
      key: 'name',
      label: 'Nhân viên',
      priority: 1,
      render: (row: Employee) => (
        <div className="flex items-center gap-3">
          <UserAvatar
            name={row.name}
            email={row.email}
            src={row.avatar}
            size="sm"
          />
          <div>
            <p className="typography-body-medium">{row.name}</p>
            <p className="typography-body-small text-muted-foreground">{row.position}</p>
          </div>
        </div>
      )
    },
    {
      key: 'project',
      label: 'Dự án',
      priority: 1,
      render: (row: Employee) => (
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 bg-primary rounded-full"></div>
          <span className="typography-body-medium text-foreground">{row.project}</span>
        </div>
      )
    },
    {
      key: 'status',
      label: 'Trạng thái',
      priority: 1,
      render: (row: Employee) => getStatusBadge(row.status),
    },
    {
      key: 'email',
      label: 'Email',
      priority: 2,
      render: (row: Employee) => (
        <div className="flex items-center gap-2">
          <div className="p-1 bg-primary/10 rounded">
            <Mail className="h-3 w-3 text-primary" />
          </div>
          <span className="typography-body-medium text-foreground">{row.email}</span>
        </div>
      )
    },
    {
      key: 'phone',
      label: 'Điện thoại',
      priority: 2,
      render: (row: Employee) => (
        <div className="flex items-center gap-2">
          <div className="p-1 bg-financial-approved/10 rounded">
            <Phone className="h-3 w-3 text-financial-approved" />
          </div>
          <span className="typography-body-medium text-foreground">{row.phone}</span>
        </div>
      )
    }
  ];

  return (
    <Card className="bg-gradient-card border border-border shadow-card">
      <CardHeader>
        <CardTitle className="typography-title-large text-foreground">Danh sách nhân viên</CardTitle>
        <CardDescription className="typography-body-medium text-muted-foreground">
          Thông tin chi tiết về các nhân viên đang tham gia dự án
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0 sm:p-6">
        <ResponsiveTable
          data={employees}
          columns={columns}
          searchKey="name"
          searchPlaceholder="Tìm kiếm nhân viên..."
          mobileFields={mobileFields}
          rowTitle={(row: Employee) => (
            <div className="flex items-center gap-3">
              <UserAvatar
                name={row.name}
                email={row.email}
                src={row.avatar}
                size="md"
              />
              <div>
                <p className="typography-title-medium text-foreground">{row.name}</p>
                <p className="typography-body-medium text-muted-foreground">{row.position}</p>
              </div>
            </div>
          )}
          rowSubtitle={(row: Employee) => (
            <div className="flex items-center gap-4 typography-body-medium text-muted-foreground">
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 bg-primary rounded-full"></div>
                <span>{row.project}</span>
              </div>
              <div className="flex items-center gap-1">
                <Mail className="w-3 h-3" />
                <span className="truncate-mobile">{row.email}</span>
              </div>
            </div>
          )}
          getRowId={(row: Employee) => String(row.id)}
          emptyState={
            <div className="text-center py-12">
              <Users className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h3 className="mt-4 typography-title-large">Không có nhân viên nào</h3>
              <p className="mt-2 typography-body-medium text-muted-foreground">
                Danh sách nhân viên sẽ hiển thị ở đây.
              </p>
            </div>
          }
          className="px-4 sm:px-0"
        />
      </CardContent>
    </Card>
  );
};
