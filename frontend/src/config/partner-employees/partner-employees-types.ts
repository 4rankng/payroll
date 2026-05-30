export interface PartnerEmployee {
  id: number;
  name: string;
  email: string;
  phone: string;
  position: string;
  project: string;
  status: string;
  current_project?: {
    project_id: number;
    project_name: string;
    project_code: string;
    client_name: string;
    position: string;
    start_date: string;
    status: string;
  };
}

export interface PartnerEmployeeStats {
  totalEmployees: number;
  activeEmployees: number;
  uniqueProjects: number;
}
