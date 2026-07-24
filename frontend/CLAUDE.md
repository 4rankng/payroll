# CLAUDE.md - Payroll Frontend

> **Long-Term Memory for AI Agents**
> This file provides context, patterns, and commands for working with the payroll frontend.

---

## 📋 Project State

### ✅ Recently Completed

**Current Status:** Stable production-ready frontend with PWA support

1. **Initial Release (v1.0.0)**
   - Full-featured payroll management system
   - Three roles: Admin, Partner, Employee
   - PWA capabilities for mobile
   - Navy & Gold premium design system

### 🎯 Next Steps

1. **UI/UX Improvements**
   - Review and fix any UI inconsistencies
   - Improve mobile responsiveness
   - Add loading states for async operations

2. **Testing**
   - Increase test coverage (currently minimal)
   - Add E2E tests with Playwright
   - Add component tests with Vitest

3. **Performance**
   - Optimize bundle size
   - Implement lazy loading for routes
   - Add image optimization

4. **Accessibility**
   - Audit with Lighthouse
   - Fix accessibility issues
   - Add ARIA labels where needed

---

## 🎨 Style Guide

### Tech Stack

**Core Technologies:**
- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool & dev server
- **Tailwind CSS** - Utility-first CSS
- **shadcn/ui** - Component library
- **React Router v6** - Routing
- **Axios** - HTTP client
- **Zustand** - State management (if used)
- **React Query** - Data fetching (if used)

### Design System

**Color Palette (TingTing Emerald):**
```css
/* Primary - TingTing Emerald */
--color-primary: #08783e;
--color-primary-light: #eaf8f0;
--color-primary-dark: #066534;

/* Secondary - Gold */
--color-secondary: #f59e0b;     /* Gold */
--color-secondary-light: #fbbf24;/* Light gold */
--color-secondary-dark: #d97706; /* Dark gold */

/* Neutral */
--color-bg: #ffffff;
--color-bg-alt: #f5f7f9;
--color-text: #101828;
--color-text-muted: #667085;
--color-border: #e4e7ec;

/* Status */
--color-success: #10b981;
--color-warning: #f59e0b;
--color-error: #ef4444;
--color-info: #3b82f6;
```

**Typography:**
```css
/* Font Family */
--font-family: 'Inter', system-ui, -apple-system, sans-serif;

/* Font Sizes */
--text-xs: 0.75rem;    /* 12px */
--text-sm: 0.875rem;   /* 14px */
--text-base: 1rem;     /* 16px */
--text-lg: 1.125rem;   /* 18px */
--text-xl: 1.25rem;    /* 20px */
--text-2xl: 1.5rem;    /* 24px */
--text-3xl: 1.875rem;  /* 30px */
--text-4xl: 2.25rem;   /* 36px */

/* Font Weights */
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;
```

**Employee self-service typography:**

- Use the `employee-type-*` semantic classes on `/employee`; do not add arbitrary pixel font sizes in employee portal components.
- Mobile scale: metadata 12px, labels/supporting copy 13px, body/actions 14px, strong values 15–16px, section titles 18px, employee name 22px, and primary money 32px.
- Reserve the 32px role for the single primary wage/advance amount. Attendance rows, badges, bank details, map guidance, and buttons stay within the 12–18px roles.
- Use only 400, 500, 600, and 700 weights in the employee portal. Prefer borders, spacing, and surface contrast over repeated shadows.
- Employee disclosures and sheets use the same semantic classes so portalled content does not fall back to the larger admin typography scale.
- Employee self-service surfaces use the `--employee-*` semantic tokens: a neutral `#F5F7F9` canvas, white surfaces, accessible dark emerald actions, soft emerald summaries, and amber only for actionable warnings. Financial status must also include text or an icon.
- Never use initials, letter monograms, or realistic human portraits as default user avatars. Use a supplied profile image when available; otherwise use the neutral abstract profile glyph.
- Keep employee identity visuals industry- and occupation-neutral. Do not infer uniforms, equipment, roles, or workplace imagery from backend domain context.

**Spacing:**
```css
--spacing-1: 0.25rem;  /* 4px */
--spacing-2: 0.5rem;   /* 8px */
--spacing-3: 0.75rem;  /* 12px */
--spacing-4: 1rem;     /* 16px */
--spacing-6: 1.5rem;   /* 24px */
--spacing-8: 2rem;     /* 32px */
--spacing-12: 3rem;    /* 48px */
--spacing-16: 4rem;    /* 64px */
```

**Border Radius:**
```css
--radius-sm: 0.25rem;  /* 4px */
--radius-md: 0.5rem;   /* 8px */
--radius-lg: 0.75rem;  /* 12px */
--radius-xl: 1rem;     /* 16px */
--radius-full: 9999px;
```

### Component Patterns

**1. Page Layout:**
```tsx
import { PageHeader } from '@/components/layout/PageHeader';
import { Container } from '@/components/layout/Container';

export function MyPage() {
  return (
    <div className="min-h-screen bg-gray-50">
      <PageHeader title="Page Title" />
      <Container className="py-6">
        {/* Page content */}
      </Container>
    </div>
  );
}
```

**2. Data Table:**
```tsx
import { DataTable } from '@/components/ui/data-table';
import { columns } from './columns';

export function MyTable({ data }) {
  return (
    <DataTable
      columns={columns}
      data={data}
      searchKey="name"
    />
  );
}
```

**3. Form with Validation:**
```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const formSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email'),
});

export function MyForm() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const onSubmit = async (data) => {
    // Submit logic
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        {/* Form fields */}
      </form>
    </Form>
  );
}
```

**4. API Call with Loading/Error:**
```tsx
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export function MyComponent() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['employees'],
    queryFn: () => api.employees.list(),
  });

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;

  return <div>{/* Render data */}</div>;
}
```

**5. Modal/Dialog:**
```tsx
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

export function MyDialog({ open, onClose }) {
  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Dialog Title</DialogTitle>
        </DialogHeader>
        {/* Dialog content */}
      </DialogContent>
    </Dialog>
  );
}
```

### Code Conventions

**File Naming:**
- Use `PascalCase` for components: `EmployeeTable.tsx`
- Use `camelCase` for utilities: `formatDate.ts`
- Use `kebab-case` for folders: `employee-management/`
- Index files: `index.ts` for barrel exports

**Component Structure:**
```tsx
// 1. Imports
import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';

// 2. Types/Interfaces
interface MyComponentProps {
  title: string;
  onAction: () => void;
}

// 3. Component
export function MyComponent({ title, onAction }: MyComponentProps) {
  // 4. Hooks
  const [state, setState] = useState(null);

  // 5. Event handlers
  const handleClick = () => {
    onAction();
  };

  // 6. Effects
  useEffect(() => {
    // Effect logic
  }, []);

  // 7. Render
  return (
    <div>
      <h1>{title}</h1>
      <Button onClick={handleClick}>Action</Button>
    </div>
  );
}
```

**Import Order:**
```tsx
// 1. React & Next.js
import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

// 2. Third-party libraries
import { z } from 'zod';
import { useQuery } from '@tanstack/react-query';

// 3. Internal components
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';

// 4. Utilities
import { formatDate } from '@/lib/utils';
import { api } from '@/lib/api';

// 5. Types
import type { Employee } from '@/types';
```

**CSS Classes:**
- Use Tailwind utility classes
- No inline styles (unless absolutely necessary)
- No CSS files (use Tailwind or CSS-in-JS)
- Use `@apply` in `index.css` for reusable patterns

**Type Safety:**
- Always define interfaces for props
- Use TypeScript for all new code
- Enable strict mode in `tsconfig.json`
- Use `zod` for runtime validation

### Linting Rules

**ESLint Configuration:**
```javascript
module.exports = {
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'prettier',
  ],
  rules: {
    'react/react-in-jsx-scope': 'off',
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',
  },
};
```

**Prettier Configuration:**
```javascript
module.exports = {
  semi: false,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
  printWidth: 100,
};
```

**Run linter:**
```bash
pnpm lint
```

**Fix linting issues:**
```bash
pnpm lint:fix
```

### Git Conventions

**Commit Message Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `style`: Code style changes (formatting)
- `perf`: Performance improvement
- `test`: Adding tests
- `docs`: Documentation
- `chore`: Maintenance tasks
- `ui`: UI/UX changes

**Examples:**
```
feat(employee): add employee import functionality

fix(timesheet): correct date parsing in export

ui(dashboard): improve responsive layout

refactor(api): consolidate API calls
```

---

## 🛠️ Command Cheat Sheet

### Development

**Install dependencies:**
```bash
pnpm install
```

**Start development server:**
```bash
pnpm dev
```
Open http://localhost:5173

**Build for production:**
```bash
pnpm build
```

**Preview production build:**
```bash
pnpm preview
```

### Testing

**Run unit tests:**
```bash
pnpm test
```

**Run tests in watch mode:**
```bash
pnpm test:watch
```

**Run tests with coverage:**
```bash
pnpm test:coverage
```

**Run E2E tests (Playwright):**
```bash
pnpm test:e2e
```

**Run E2E tests with UI:**
```bash
pnpm test:e2e:ui
```

**Run specific test:**
```bash
pnpm test EmployeeTable
```

### Code Quality

**Run linter:**
```bash
pnpm lint
```

**Fix linting issues:**
```bash
pnpm lint:fix
```

**Type check:**
```bash
pnpm type-check
```

**Format code:**
```bash
pnpm format
```

**Run all checks:**
```bash
pnpm check
```

### Dependencies

**Update dependencies:**
```bash
pnpm update
```

**Check for outdated packages:**
```bash
pnpm outdated
```

**Add a new package:**
```bash
pnpm add package-name
```

**Add a dev dependency:**
```bash
pnpm add -D package-name
```

### Build & Deploy

**Build for development:**
```bash
pnpm build:dev
```

**Build for production:**
```bash
pnpm build
```

**Analyze bundle size:**
```bash
pnpm build:analyze
```

**Generate PWA assets:**
```bash
pnpm pwa:generate
```

### Useful Aliases (add to ~/.bashrc or ~/.zshrc)

```bash
# Payroll Frontend aliases
alias pf-dev='cd ~/payroll-frontend && pnpm dev'
alias pf-build='cd ~/payroll-frontend && pnpm build'
alias pf-test='cd ~/payroll-frontend && pnpm test'
alias pf-lint='cd ~/payroll-frontend && pnpm lint:fix'
alias pf-preview='cd ~/payroll-frontend && pnpm preview'
alias pf-install='cd ~/payroll-frontend && pnpm install'
```

---

## 📚 Key Files & Patterns

### Important Files

- **`src/main.tsx`** - Application entry point
- **`src/App.tsx`** - Main app component with routing
- **`src/router/index.tsx`** - Route definitions
- **`src/lib/api.ts`** - API client configuration
- **`src/lib/utils.ts`** - Utility functions
- **`tailwind.config.ts`** - Tailwind configuration
- **`vite.config.ts`** - Vite configuration
- **`tsconfig.json`** - TypeScript configuration
- **`.env.example`** - Environment variables template

### Directory Structure

```
src/
├── components/          # Reusable components
│   ├── ui/             # shadcn/ui components
│   ├── layout/         # Layout components
│   └── features/       # Feature-specific components
├── pages/              # Page components
├── lib/                # Utilities & configurations
├── hooks/              # Custom React hooks
├── stores/             # State management (Zustand)
├── types/              # TypeScript type definitions
├── services/           # API services
├── constants/          # Constants & enums
└── assets/             # Static assets
```

### Common Patterns

**1. Creating a New Page:**
```tsx
// src/pages/employee/EmployeeList.tsx
import { PageHeader } from '@/components/layout/PageHeader';
import { DataTable } from '@/components/ui/data-table';
import { columns } from './EmployeeList.columns';

export function EmployeeListPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['employees'],
    queryFn: () => api.employees.list(),
  });

  return (
    <div className="min-h-screen bg-gray-50">
      <PageHeader
        title="Employees"
        actions={<Button>Add Employee</Button>}
      />
      <Container className="py-6">
        {isLoading ? <LoadingSpinner /> : <DataTable columns={columns} data={data} />}
      </Container>
    </div>
  );
}
```

**2. Creating a New Component:**
```tsx
// src/components/features/EmployeeCard.tsx
import { Card, CardHeader, CardContent } from '@/components/ui/card';
import type { Employee } from '@/types';

interface EmployeeCardProps {
  employee: Employee;
}

export function EmployeeCard({ employee }: EmployeeCardProps) {
  return (
    <Card>
      <CardHeader>
        <h3>{employee.name}</h3>
      </CardHeader>
      <CardContent>
        <p>{employee.email}</p>
      </CardContent>
    </Card>
  );
}
```

**3. Using shadcn/ui Components:**
```tsx
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';

export function MyComponent() {
  return (
    <div>
      <Button>Click me</Button>
      <Input placeholder="Enter text" />
      <Dialog>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Dialog Title</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    </div>
  );
}
```

**4. API Service:**
```tsx
// src/services/employee.ts
import { api } from '@/lib/api';
import type { Employee, CreateEmployeeInput } from '@/types';

export const employeeService = {
  list: () => api.get<Employee[]>('/employees'),
  get: (id: number) => api.get<Employee>(`/employees/${id}`),
  create: (data: CreateEmployeeInput) => api.post<Employee>('/employees', data),
  update: (id: number, data: Partial<CreateEmployeeInput>) =>
    api.put<Employee>(`/employees/${id}`, data),
  delete: (id: number) => api.delete(`/employees/${id}`),
};
```

**5. Custom Hook:**
```tsx
// src/hooks/useEmployees.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { employeeService } from '@/services/employee';

export function useEmployees() {
  return useQuery({
    queryKey: ['employees'],
    queryFn: employeeService.list,
  });
}

export function useCreateEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: employeeService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
    },
  });
}
```

**6. Route Configuration:**
```tsx
// src/router/index.tsx
import { createBrowserRouter } from 'react-router-dom';
import { Layout } from '@/components/layout/Layout';
import { EmployeeListPage } from '@/pages/employee/EmployeeList';
import { EmployeeDetailPage } from '@/pages/employee/EmployeeDetail';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { path: 'employees', element: <EmployeeListPage /> },
      { path: 'employees/:id', element: <EmployeeDetailPage /> },
    ],
  },
]);
```

---

## 🔍 Troubleshooting

### Common Issues

**1. Development server not starting:**
```bash
# Clear node_modules and reinstall
rm -rf node_modules pnpm-lock.yaml
pnpm install

# Check if port 5173 is in use
lsof -i :5173
# Or use a different port
pnpm dev --port 3000
```

**2. Build errors:**
```bash
# Check TypeScript errors
pnpm type-check

# Check for unused imports
pnpm lint

# Clear Vite cache
rm -rf node_modules/.vite
pnpm build
```

**3. API connection issues:**
```bash
# Check .env file
cat .env

# Verify API URL is correct
echo $VITE_API_BASE_URL

# Check if backend is running
curl http://localhost:8080/api/v1/health
```

**4. Styling not applying:**
```bash
# Check Tailwind config
cat tailwind.config.ts

# Verify Tailwind directives in index.css
cat src/index.css | grep -i tailwind

# Restart dev server
```

**5. Tests failing:**
```bash
# Run tests in verbose mode
pnpm test --verbose

# Run specific test file
pnpm test EmployeeList.test.tsx

# Update snapshots
pnpm test -u
```

---

## 📖 Additional Resources

### Documentation
- [React Documentation](https://react.dev/)
- [TypeScript Documentation](https://www.typescriptlang.org/docs/)
- [Vite Documentation](https://vitejs.dev/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [shadcn/ui](https://ui.shadcn.com/)
- [React Router](https://reactrouter.com/)
- [TanStack Query](https://tanstack.com/query/latest)
- [Playwright](https://playwright.dev/)

### Internal Documentation
- `README.md` - Project overview
- `DESIGN-GUIDE.md` - Design system guidelines
- `UI-UX-REVIEW.md` - UI/UX review notes
- `QA.md` - QA guidelines

---

## 🔄 Updating This File

**When to update CLAUDE.md:**

1. **After completing major features:** Update "Project State" section
2. **After architectural changes:** Update "Architecture Patterns" section
3. **After adding new commands:** Update "Command Cheat Sheet" section
4. **After changing conventions:** Update "Style Guide" section
5. **After discovering new patterns:** Add to "Common Patterns" section

**Template for updates:**

```markdown
### ✅ Recently Completed (YYYY-MM-DD)

1. **Feature Name** - Status
   - Key files changed
   - Important notes

### 🎯 Next Steps

1. **Next Feature** - Priority
   - Description
```

---

**Last Updated:** 2026-05-06
**Maintained By:** Orbit (AI Agent)
**Version:** 1.0


QA Testing

localhost:3000

admin login
frankng
Admin123

partner login
cuongnv
Admin123

all accounts have password Admin123
