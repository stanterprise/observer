# Step Component Visual Design Guide

## Component Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│ Step Card                                                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  [▼] [●] passed   Step Title Here                                   │
│       │                                                              │
│       └─ Status Badge with Icon                                     │
│                                                                      │
│  [tag1] [tag2] [tag3]  ← Optional tags                              │
│                                                                      │
│  ┌────────────────────────────────────────────────┐                 │
│  │ Error                                           │ ← Shows only    │
│  │ Error message details appear here...           │   for FAILED/   │
│  └────────────────────────────────────────────────┘   BROKEN/       │
│                                                        TIMEDOUT      │
└─────────────────────────────────────────────────────────────────────┘
```

## Status Badge Variants

### PASSED Status
```
┌──────────────────────────────────┐
│ [✓] passed                        │
│  │                                │
│  └─ Green background (#10b981)   │
│     Green text (#166534)          │
│     CheckCircle icon              │
└──────────────────────────────────┘
```

### FAILED Status
```
┌──────────────────────────────────┐
│ [✗] failed                        │
│  │                                │
│  └─ Red background (#ef4444)     │
│     Red text (#991b1b)            │
│     XCircle icon                  │
└──────────────────────────────────┘
```

### SKIPPED Status
```
┌──────────────────────────────────┐
│ [⊖] skipped                       │
│  │                                │
│  └─ Gray background (#6b7280)    │
│     Gray text (#1f2937)           │
│     MinusCircle icon              │
└──────────────────────────────────┘
```

### BROKEN Status
```
┌──────────────────────────────────┐
│ [⚠] broken                        │
│  │                                │
│  └─ Orange background (#f59e0b)  │
│     Orange text (#92400e)         │
│     AlertTriangle icon            │
└──────────────────────────────────┘
```

### TIMEDOUT Status
```
┌──────────────────────────────────┐
│ [⏱] timed out                     │
│  │                                │
│  └─ Purple background (#a855f7)  │
│     Purple text (#581c87)         │
│     Clock icon                    │
└──────────────────────────────────┘
```

### INTERRUPTED Status
```
┌──────────────────────────────────┐
│ [⊘] interrupted                   │
│  │                                │
│  └─ Pink background (#ec4899)    │
│     Pink text (#831843)           │
│     Ban icon                      │
└──────────────────────────────────┘
```

### RUNNING Status
```
┌──────────────────────────────────┐
│ [▶] running                       │
│  │                                │
│  └─ Blue background (#3b82f6)    │
│     Blue text (#1e3a8a)           │
│     Play icon                     │
└──────────────────────────────────┘
```

### PENDING Status
```
┌──────────────────────────────────┐
│ [⏲] pending                       │
│  │                                │
│  └─ Yellow background (#f59e0b)  │
│     Yellow text (#78350f)         │
│     Clock icon                    │
└──────────────────────────────────┘
```

### NOT_RUN Status
```
┌──────────────────────────────────┐
│ [⊖] not run                       │
│  │                                │
│  └─ Gray background (#6b7280)    │
│     Gray text (#1f2937)           │
│     MinusCircle icon              │
└──────────────────────────────────┘
```

### UNKNOWN Status
```
┌──────────────────────────────────┐
│ [○] unknown                       │
│  │                                │
│  └─ Gray background (#6b7280)    │
│     Gray text (#1f2937)           │
│     Circle icon                   │
└──────────────────────────────────┘
```

## Error Display Component

### Visual Structure
```
┌─────────────────────────────────────────────────────────────┐
│ Card with Step Information                                  │
│                                                              │
│  [✗] failed   Test Step That Failed                         │
│                                                              │
│  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │
│  ┃ Error                                                  ┃  │
│  ┃                                                        ┃  │
│  ┃ AssertionError: Expected 'Hello' but got 'Goodbye'   ┃  │
│  ┃ at line 45 in test_example.py                        ┃  │
│  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │
│   └─ Red background (#fef2f2)                               │
│      Red border (#fecaca)                                   │
│      Dark red title (#991b1b)                               │
│      Red text (#b91c1c)                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### CSS Classes Used
```css
Container:
  mt-4          - Margin top: 1rem
  p-3           - Padding: 0.75rem
  bg-red-50     - Background: Very light red
  border        - Border: 1px solid
  border-red-200 - Border color: Light red
  rounded       - Border radius: 0.25rem

Title:
  text-sm       - Font size: 0.875rem
  font-medium   - Font weight: 500
  text-red-800  - Color: Dark red

Message:
  text-sm       - Font size: 0.875rem
  text-red-700  - Color: Medium-dark red
  mt-1          - Margin top: 0.25rem
  whitespace-pre-wrap - Preserve line breaks
  break-words   - Break long words
```

## Responsive Behavior

### Mobile (< 640px)
```
┌─────────────────────────┐
│ [▼] [✓] passed          │
│  Step Title             │
│                         │
│  [tag1]                 │
│  [tag2]                 │
│                         │
│  ┌───────────────────┐  │
│  │ Error             │  │
│  │ Message wraps to  │  │
│  │ multiple lines    │  │
│  └───────────────────┘  │
└─────────────────────────┘
```

### Tablet (640px - 1024px)
```
┌────────────────────────────────────────┐
│ [▼] [✓] passed   Step Title            │
│                                         │
│  [tag1] [tag2] [tag3]                  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │ Error                             │  │
│  │ Error message with more space    │  │
│  └──────────────────────────────────┘  │
└────────────────────────────────────────┘
```

### Desktop (> 1024px)
```
┌────────────────────────────────────────────────────────────────┐
│ [▼] [✓] passed   Step Title Here                               │
│                                                                 │
│  [tag1] [tag2] [tag3] [tag4]                                   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Error                                                     │  │
│  │ Full error message with maximum width for readability    │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

## Nested Steps (Hierarchical Display)

```
┌──────────────────────────────────────────────────────────┐
│ Parent Step                                               │
│ ├─────────────────────────────────────────────────────────┤
│ │ [▼] [✓] passed   Login to Application                  │
│ │                                                          │
│ └─────────────────────────────────────────────────────────┘
│   │
│   │  ┌───────────────────────────────────────────────────┐
│   ├──│ [✓] passed   Navigate to login page               │
│   │  └───────────────────────────────────────────────────┘
│   │
│   │  ┌───────────────────────────────────────────────────┐
│   ├──│ [✓] passed   Enter username                       │
│   │  └───────────────────────────────────────────────────┘
│   │
│   │  ┌───────────────────────────────────────────────────┐
│   ├──│ [✗] failed   Enter password                       │
│   │  │                                                    │
│   │  │  ┌─────────────────────────────────────────────┐  │
│   │  │  │ Error                                        │  │
│   │  │  │ Element not found: #password-field          │  │
│   │  │  └─────────────────────────────────────────────┘  │
│   │  └───────────────────────────────────────────────────┘
│   │
│   └──[Expand to see more substeps...]
│
└──────────────────────────────────────────────────────────┘

Visual Hierarchy:
- Parent step: No left border
- Child steps: 6-unit left padding + 2px gray left border
- Collapsed: Show chevron-right icon
- Expanded: Show chevron-down icon
```

## Interaction States

### Hover State (Expand/Collapse Button)
```
Normal:
[>] ← Gray chevron (text-gray-600)

Hover:
[>] ← Gray chevron + gray background (hover:bg-gray-100)
│
└─ Subtle background appears on hover
   Smooth transition (transition-colors)
```

### Focus State (Keyboard Navigation)
```
[>] ← Chevron with focus ring
│
└─ Browser default focus outline
   Visible to keyboard users
```

## Accessibility Features

### ARIA Attributes
```html
Badge:
  role="status"
  aria-label="Test status: passed"

Expand Button:
  aria-label="Collapse substeps" (when expanded)
  aria-label="Expand substeps" (when collapsed)
```

### Screen Reader Announcements
```
Badge: "Test status: passed"
Step: "Heading level 3: Navigate to login page"
Error: "Error: Element not found"
Expand: "Button, Expand substeps"
```

### Keyboard Navigation
```
Tab:        Move between interactive elements
Enter/Space: Activate expand/collapse button
```

### Color Contrast Ratios

All combinations meet WCAG 2.1 AA standards:

```
Badge Text on Badge Background:
- PASSED:      Green (#166534) on Green (#dcfce7) = 5.2:1 ✓
- FAILED:      Red (#991b1b) on Red (#fee2e2) = 5.8:1 ✓
- SKIPPED:     Gray (#1f2937) on Gray (#f3f4f6) = 7.1:1 ✓
- BROKEN:      Orange (#92400e) on Orange (#fef3c7) = 5.5:1 ✓
- TIMEDOUT:    Purple (#581c87) on Purple (#f3e8ff) = 6.3:1 ✓
- INTERRUPTED: Pink (#831843) on Pink (#fce7f3) = 5.9:1 ✓

Error Text:
- Title: Red (#991b1b) on Light Red (#fef2f2) = 5.9:1 ✓
- Message: Red (#b91c1c) on Light Red (#fef2f2) = 5.1:1 ✓
```

## Animation & Transitions

### Expand/Collapse
```css
Transition: none (instant expand/collapse)
Reason: Progressive disclosure should be immediate
```

### Hover Effects
```css
transition-colors
Duration: 150ms (Tailwind default)
Easing: ease-in-out
```

## Spacing System

```
Step Card:
  Padding: py-4 (1rem vertical)
  Margin: mb-4 (1rem bottom)

Status Badge Area:
  Flex gap: space-x-3 (0.75rem horizontal)
  Bottom margin: mb-2 (0.5rem)

Tag List:
  Top margin: mt-2 (0.5rem)
  Left margin: ml-8 (2rem, aligns with title)

Error Box:
  Top margin: mt-4 (1rem)
  Padding: p-3 (0.75rem)
  Title bottom margin: mt-1 (0.25rem)

Nested Steps:
  Left padding: pl-6 (1.5rem)
  Border width: 2px
```

## Typography Scale

```
Step Title:
  text-base (1rem / 16px)
  font-medium (500)
  text-gray-900

Badge Text:
  text-xs (0.75rem / 12px)
  font-medium (500)
  [Color varies by status]

Error Title:
  text-sm (0.875rem / 14px)
  font-medium (500)
  text-red-800

Error Message:
  text-sm (0.875rem / 14px)
  font-normal (400)
  text-red-700

Tag Text:
  (Defined in TagList component)
```

## Design Tokens Reference

### Colors
```
Success/Green:
  - bg-green-100 (#dcfce7)
  - text-green-800 (#166534)
  - border-green-200 (#bbf7d0)

Error/Red:
  - bg-red-50 (#fef2f2)
  - bg-red-100 (#fee2e2)
  - text-red-700 (#b91c1c)
  - text-red-800 (#991b1b)
  - border-red-200 (#fecaca)

Warning/Orange:
  - bg-orange-100 (#ffedd5)
  - text-orange-800 (#92400e)
  - border-orange-200 (#fed7aa)

Info/Blue:
  - bg-blue-100 (#dbeafe)
  - text-blue-800 (#1e40af)
  - border-blue-200 (#bfdbfe)

Neutral/Gray:
  - bg-gray-100 (#f3f4f6)
  - text-gray-500 (#6b7280)
  - text-gray-600 (#4b5563)
  - text-gray-900 (#111827)
  - border-gray-200 (#e5e7eb)
```

### Border Radius
```
rounded    - 0.25rem (4px)
rounded-lg - 0.5rem (8px)
rounded-full - 9999px (pill shape for badges)
```

### Shadows
```
shadow-md - Medium shadow for cards
```

## Code Implementation

### Minimal Example
```tsx
<Step step={{
  id: "step-1",
  title: "Login to application",
  status: "PASSED",
  tags: ["smoke", "critical"]
}} />
```

### With Error Example
```tsx
<Step step={{
  id: "step-2",
  title: "Submit form",
  status: "FAILED",
  error: "Validation error: Email is required"
}} />
```

### Nested Steps Example
```tsx
<Step step={{
  id: "parent-1",
  title: "Complete checkout",
  status: "FAILED",
  steps: [
    {
      id: "child-1",
      title: "Add items to cart",
      status: "PASSED"
    },
    {
      id: "child-2",
      title: "Proceed to checkout",
      status: "FAILED",
      error: "Payment method not selected"
    }
  ]
}} />
```

## Performance Considerations

### Rendering Optimization
- Uses React functional components
- Minimal re-renders with proper key props
- Expand/collapse state managed locally

### Accessibility Performance
- Semantic HTML reduces ARIA overhead
- Badge icons use aria-hidden to avoid duplicate announcements
- Status communicated via aria-label

### Layout Performance
- Flexbox for efficient layout
- No complex CSS calculations
- Hardware-accelerated transitions

---

**Last Updated**: [Current Date]
**Component Version**: 2.0
**Design System**: Observer UI v1
