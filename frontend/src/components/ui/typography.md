# Typography System Documentation

## Overview
Our application uses a comprehensive typography system based on Material Design 3 principles, ensuring consistent text styling across all components, pages, modals, and sheets.

## Design Philosophy
- **Semantic Typography**: Use semantic tokens instead of arbitrary sizes
- **Accessibility First**: Ensure proper contrast ratios and readability
- **Responsive Design**: Typography scales appropriately across devices
- **Systematic Approach**: Consistent patterns for all text types

## Typography Variants

### Display Typography (Large headlines, hero sections)
- `display-large`: 24-28px, bold, tight spacing - for hero headlines
- `display-medium`: 20-24px, bold, tight spacing - for page headers
- `display-small`: 18-20px, bold, snug spacing - for section headers

### Headlines (Section titles, card headers)
- `headline-large`: 16px, semibold, snug spacing - for major section titles
- `headline-medium`: 15px, semibold, snug spacing - for subsection titles
- `headline-small`: 14px, semibold, normal spacing - for card titles

### Titles (Component labels, form titles)
- `title-large`: 14px, medium, normal spacing - for large component titles
- `title-medium`: 13px, medium, normal spacing - for medium component titles
- `title-small`: 12px, medium, wide spacing - for small component titles

### Body Text (Content, descriptions)
- `body-large`: 13px, normal, relaxed spacing - for primary content
- `body-medium`: 12px, normal, relaxed spacing - for secondary content
- `body-small`: 11px, normal, normal spacing - for captions and metadata

### Labels (Form labels, navigation, status)
- `label-large`: 12px, medium, wide spacing - for form labels
- `label-medium`: 11px, medium, wide spacing - for navigation items
- `label-small`: 11px, medium, wider spacing, uppercase - for badges/status

## Usage Examples

### Using Typography Components
```tsx
import { PageTitle, PageSubtitle, CardTitle, Body } from "@/components/ui/typography";

// Page headers
<PageTitle>Dự án</PageTitle>
<PageSubtitle>Theo dõi và quản lý tất cả dự án trong công ty</PageSubtitle>

// Card content
<CardTitle>Thông tin dự án</CardTitle>
<Body>Chi tiết về dự án và tiến độ thực hiện</Body>
```

### Using Typography Utility Classes
```tsx
// Direct CSS classes
<h1 className="typography-display-medium">Page Title</h1>
<p className="typography-body-medium text-muted-foreground">Description</p>

// With Typography component
<Typography variant="headline-large" color="primary">
  Section Title
</Typography>
```

### Using Typography Presets
```tsx
import { useTypographyPreset } from "@/components/ui/typography";

const preset = useTypographyPreset("pageTitle");
// Returns: { variant: "display-medium", color: "primary" }
```

## Color Variants
- `primary`: Default foreground color
- `secondary`: Secondary foreground color
- `muted`: Muted/subdued text
- `accent`: Accent color text
- `destructive`: Error/warning text
- `success`: Success state text
- `warning`: Warning state text
- `info`: Information text

## Component Mapping

### Page Headers
```tsx
// Before
<h1 className="text-2xl md:text-3xl font-bold">Title</h1>
<p className="text-muted-foreground">Description</p>

// After
<PageTitle>Title</PageTitle>
<PageSubtitle>Description</PageSubtitle>
```

### Modal/Sheet Headers
```tsx
// Use typography presets
<Typography {...useTypographyPreset("modalTitle")}>
  Modal Title
</Typography>
<Typography {...useTypographyPreset("modalDescription")}>
  Modal description text
</Typography>
```

### Form Elements
```tsx
// Form labels
<Label {...useTypographyPreset("formLabel")}>
  Tên dự án
</Label>

// Helper text
<Typography {...useTypographyPreset("formHelper")}>
  Nhập tên dự án của bạn
</Typography>

// Error messages
<Typography {...useTypographyPreset("formError")}>
  Trường này là bắt buộc
</Typography>
```

### Table Content
```tsx
// Table headers
<th className={getTypographyClasses({
  variant: "label-large",
  color: "medium-contrast"
})}>
  Header
</th>

// Table cells
<td className={getTypographyClasses({
  variant: "body-medium",
  color: "primary"
})}>
  Content
</td>
```

## CSS Variables Reference

### Typography Scales
- `--text-xs`: 0.6875rem (11px)
- `--text-sm`: 0.75rem (12px)
- `--text-base`: 0.8125rem (13px)
- `--text-lg`: 0.875rem (14px)
- `--text-xl`: 1rem (16px)
- `--text-2xl`: 1.125rem (18px)
- `--text-3xl`: 1.25rem (20px)
- `--text-4xl`: 1.5rem (24px)
- `--text-5xl`: 1.75rem (28px)

### Line Heights
- `--leading-none`: 1
- `--leading-tight`: 1.15
- `--leading-snug`: 1.3
- `--leading-normal`: 1.4
- `--leading-relaxed`: 1.5
- `--leading-loose`: 1.65

### Letter Spacing
- `--tracking-tighter`: -0.05em
- `--tracking-tight`: -0.025em
- `--tracking-normal`: 0em
- `--tracking-wide`: 0.025em
- `--tracking-wider`: 0.05em
- `--tracking-widest`: 0.1em

## Migration Guide

### Replacing Hardcoded Typography
```tsx
// ❌ Before: Hardcoded classes
<h1 className="text-2xl font-bold text-slate-900">Title</h1>
<p className="text-sm text-gray-600">Description</p>

// ✅ After: Semantic typography
<H1>Title</H1>
<Caption>Description</Caption>

// Or with direct classes
<h1 className="typography-display-medium">Title</h1>
<p className="typography-body-small text-muted">Description</p>
```

### Modal Headers
```tsx
// ❌ Before
<DialogTitle className="text-lg font-semibold">
  Modal Title
</DialogTitle>

// ✅ After
<DialogTitle className={getTypographyClasses({
  variant: "headline-small",
  color: "primary"
})}>
  Modal Title
</DialogTitle>
```

## Best Practices

1. **Use Semantic Components**: Prefer `<PageTitle>` over manual typography classes
2. **Consistent Hierarchy**: Follow the typography scale consistently
3. **Color Tokens**: Always use semantic color tokens, never hardcoded colors
4. **Responsive Design**: Typography automatically scales on mobile
5. **Accessibility**: Maintain proper contrast ratios using predefined color combinations

## Accessibility Features
- Proper contrast ratios for all color combinations
- Scalable typography that works with browser zoom
- Clear visual hierarchy with appropriate spacing
- Support for screen readers with semantic HTML

## Integration with Design System
The typography system is fully integrated with:
- Color tokens from `index.css`
- Responsive utilities from `responsive-utils.ts`
- Component variants in shadcn/ui components
- Theme switching (light/dark mode)

## Future Enhancements
- Font loading optimization
- Variable font support
- Enhanced mobile typography scaling
- Custom typography themes for different user roles
