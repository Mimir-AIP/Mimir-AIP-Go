# Mimir AIP Frontend Design Specification

## Brand Identity

**Mimir AIP** is a powerful, enterprise-grade AI orchestration platform inspired by Norse mythology (Mimir - the god of wisdom). The design should convey:
- **Intelligence & Wisdom** - Clean, data-focused interfaces
- **Power & Reliability** - Solid, trustworthy aesthetics
- **Modern & Cutting-edge** - Contemporary design patterns

---

## Color Palette

### Primary Colors
| Name | Hex | Usage |
|------|-----|-------|
| Navy | `#0B192C` | Primary background, deep sections |
| Blue | `#1E3E62` | Cards, secondary backgrounds, borders |
| Orange | `#FF6500` | Accent, CTAs, highlights, active states |
| White | `#FFFFFF` | Text, icons on dark backgrounds |

### Secondary Colors
| Name | Hex | Usage |
|------|-----|-------|
| Success | `#22C55E` | Success states, positive metrics |
| Warning | `#F59E0B` | Warnings, pending states |
| Error | `#EF4444` | Errors, destructive actions |
| Info | `#3B82F6` | Informational elements |

### Gradients
```css
/* Hero gradient */
background: linear-gradient(135deg, #0B192C 0%, #1E3E62 50%, #0B192C 100%);

/* Accent gradient for important CTAs */
background: linear-gradient(135deg, #FF6500 0%, #FF8C00 100%);

/* Subtle card gradient */
background: linear-gradient(180deg, #1E3E62 0%, #152D4A 100%);
```

---

## Typography

### Font Stack
- **Primary**: `Geist Sans` - Modern, clean, excellent readability
- **Monospace**: `Geist Mono` - Code, IDs, technical content

### Type Scale
| Element | Size | Weight | Line Height |
|---------|------|--------|-------------|
| H1 | 2.25rem (36px) | 700 | 1.2 |
| H2 | 1.5rem (24px) | 600 | 1.3 |
| H3 | 1.25rem (20px) | 600 | 1.4 |
| Body | 1rem (16px) | 400 | 1.5 |
| Small | 0.875rem (14px) | 400 | 1.5 |
| Tiny | 0.75rem (12px) | 400 | 1.4 |

### Heading Colors
- `H1`: Orange (`#FF6500`) - Page titles
- `H2`: Orange (`#FF6500`) - Section titles
- `H3`: White (`#FFFFFF`) - Subsection titles

---

## Spacing System

Based on 4px grid:
| Name | Value | Usage |
|------|-------|-------|
| xs | 4px | Tight spacing |
| sm | 8px | Component internal |
| md | 16px | Between elements |
| lg | 24px | Section padding |
| xl | 32px | Page margins |
| 2xl | 48px | Major sections |

---

## Component Standards

### Cards
```css
.card {
  background: #1E3E62; /* blue */
  border: 1px solid #1E3E62;
  border-radius: 0.5rem;
  padding: 1.5rem;
}

.card:hover {
  border-color: #FF6500; /* orange accent on hover */
  box-shadow: 0 4px 12px rgba(255, 101, 0, 0.15);
}
```

### Buttons

#### Primary (Orange)
```css
.btn-primary {
  background: #FF6500;
  color: #0B192C;
  font-weight: 600;
  padding: 0.5rem 1rem;
  border-radius: 0.5rem;
}
.btn-primary:hover {
  background: #FF8C00;
}
```

#### Secondary (Outline)
```css
.btn-secondary {
  background: transparent;
  border: 1px solid #1E3E62;
  color: #FFFFFF;
}
.btn-secondary:hover {
  border-color: #FF6500;
  color: #FF6500;
}
```

### Tables
```css
.table {
  background: #1E3E62;
  border-radius: 0.5rem;
  overflow: hidden;
}
.table th {
  background: #0B192C;
  color: #9CA3AF; /* gray-400 */
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}
.table tr:hover {
  background: #0B192C;
}
```

### Form Inputs
```css
.input {
  background: rgba(30, 62, 98, 0.3); /* blue/30 */
  border: 1px solid #1E3E62;
  color: #FFFFFF;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
}
.input:focus {
  border-color: #FF6500;
  outline: none;
  box-shadow: 0 0 0 2px rgba(255, 101, 0, 0.2);
}
```

### Badges/Tags
```css
/* Ingestion */
.badge-ingestion { background: #3B82F6; color: white; }

/* Processing */
.badge-processing { background: #8B5CF6; color: white; }

/* Output */
.badge-output { background: #22C55E; color: white; }

/* Status badges */
.badge-active { background: #22C55E; }
.badge-pending { background: #F59E0B; }
.badge-failed { background: #EF4444; }
.badge-completed { background: #3B82F6; }
```

---

## Layout Patterns

### Sidebar
- Width: 256px (16rem)
- Fixed position
- Collapsible on mobile
- Logo at top with brand name
- Grouped navigation with expandable sections

### Page Structure
```
┌─────────────────────────────────────────────┐
│ Topbar (Welcome, user actions)              │
├─────────────────────────────────────────────┤
│ Page Header                                  │
│ ┌─────────────────────────────────────────┐ │
│ │ H1 Title           [Action Buttons]     │ │
│ │ Description text                        │ │
│ └─────────────────────────────────────────┘ │
├─────────────────────────────────────────────┤
│ Content Area                                 │
│ ┌─────────────────────────────────────────┐ │
│ │ Cards / Tables / Forms                  │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Grid System
- Desktop: 3 columns for cards
- Tablet: 2 columns
- Mobile: 1 column
- Gap: 24px (1.5rem)

---

## Iconography

Use Lucide React icons consistently:
- Size: 16px for inline, 20px for buttons, 24px for feature icons
- Color: Inherit from text or use orange for emphasis
- Stroke width: 2

### Common Icons
| Action | Icon |
|--------|------|
| Create | Plus |
| Edit | Pencil |
| Delete | Trash2 |
| View | Eye |
| Settings | Settings |
| Refresh | RefreshCw |
| Download | Download |
| Upload | Upload |
| Pipeline | GitBranch |
| Ontology | Network |
| ML Model | Brain |
| Digital Twin | Copy |
| Workflow | Workflow |

---

## Animation & Motion

### Transitions
- Default: `transition-colors duration-200`
- Hover effects: `transition-all duration-200`
- Loading states: Pulse animation

### Page Transitions
- Fade in on load: `animate-in fade-in duration-300`

### Skeleton Loading
```css
.skeleton {
  background: linear-gradient(
    90deg,
    #1E3E62 25%,
    #2a5080 50%,
    #1E3E62 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
```

---

## Responsive Breakpoints

| Breakpoint | Width | Layout |
|------------|-------|--------|
| Mobile | < 768px | Single column, hidden sidebar |
| Tablet | 768-1024px | 2 columns, collapsible sidebar |
| Desktop | > 1024px | Full layout, fixed sidebar |

---

## Accessibility

1. **Color Contrast**: Minimum 4.5:1 ratio
2. **Focus States**: Visible orange ring on focus
3. **Keyboard Navigation**: Full tab support
4. **Screen Readers**: Proper ARIA labels
5. **Reduced Motion**: Respect `prefers-reduced-motion`

---

## Page-Specific Guidelines

### Dashboard
- Stats cards at top (3-column grid)
- Quick actions prominently displayed
- Recent activity feed
- Performance metrics visualization

### Pipelines
- Card grid for pipeline list
- Clear type badges (Ingestion/Processing/Output)
- Prominent Create button
- Status indicators

### Ontologies
- Table view for list
- "Create from Pipeline" as primary CTA
- Upload as secondary action
- Version info visible

### Workflows
- Progress visualization (step indicators)
- Real-time status updates
- Clear step-by-step UI
- Artifact links

### Chat
- Full-height chat interface
- Message bubbles with clear sender distinction
- Tool call visualization
- Context sidebar

---

## Anti-Patterns (Avoid)

1. ❌ Generic gray backgrounds
2. ❌ Low-contrast text
3. ❌ Inconsistent spacing
4. ❌ Mixed icon styles
5. ❌ Overloaded pages without visual hierarchy
6. ❌ Raw JSON dumps (format data nicely)
7. ❌ Missing loading states
8. ❌ Missing error states
9. ❌ Buttons without hover states
10. ❌ Tables without row hover highlighting

---

## Component Checklist

For each page, ensure:
- [ ] Proper H1 page title in orange
- [ ] Loading skeleton while fetching data
- [ ] Error state with retry option
- [ ] Empty state with helpful message
- [ ] Consistent card styling
- [ ] Hover states on interactive elements
- [ ] Toast notifications for actions
- [ ] Responsive layout

---

*Last Updated: December 2024*

