# Igreja Organizada — UI/UX Standards
Version: 1.0.0 · Read completely before writing any component.

> **For the frontend agent:** This is not a suggestion list. These are the standards
> that define quality in this product. Every screen you build is measured against them.

---

## 1. Design Philosophy

This is a **mobile-first PWA** used primarily on smartphones by church volunteers and pastors — often during or right before a service, in a hurry, sometimes in low-light environments.

Every design decision must answer: **"Does this work for someone using their phone with one hand, under time pressure?"**

Principles (in priority order):
1. **Clarity over cleverness** — the user must never have to think about what a UI element does
2. **Speed of task completion** — minimize taps to complete any core action
3. **Forgiveness** — destructive actions require confirmation; mistakes are recoverable where possible
4. **Consistency** — same pattern for same problem, everywhere in the app
5. **Accessibility** — the app must work for people with visual impairments and older devices

---

## 2. Layout & Spacing

### Touch targets
- Minimum touch target: **48×48px** (Apple HIG and Material Design standard)
- Never place two tappable elements closer than **8px** apart
- Primary actions (save, confirm, publish) must be reachable with the thumb — bottom half of screen on mobile

### Spacing scale
Use Tailwind's default spacing scale. Stick to multiples of 4px:
- Tight (within a component): `p-2`, `gap-2` (8px)
- Default (between related elements): `p-4`, `gap-4` (16px)
- Loose (between sections): `p-6`, `gap-6` (24px)
- Section breaks: `py-8` or `py-10`

Never use arbitrary values (`p-[13px]`) unless there is no alternative.

### Container width
- Mobile: full width with `px-4` horizontal padding
- Tablet/desktop: max-width `max-w-2xl` centered for content, `max-w-5xl` for dashboards
- Never design desktop-only layouts — mobile is the primary target

---

## 3. Typography

Use Tailwind's type scale. Rules:
- Page titles: `text-xl font-semibold` or `text-2xl font-bold`
- Section headers: `text-base font-semibold`
- Body text: `text-sm` or `text-base` — never smaller than `text-xs` for readable content
- Supporting/meta text: `text-xs text-muted-foreground`
- Never use more than **2 font sizes** on a single card or list item
- Avoid `font-light` — low contrast on small screens

Line height: default Tailwind line heights are correct. Do not override them.

---

## 4. Color & Contrast

- Minimum contrast ratio: **4.5:1** for body text (WCAG AA)
- Minimum contrast ratio: **3:1** for large text and UI components
- Never convey information by color alone — pair with icon, label, or pattern
- Use semantic color tokens from shadcn/ui (`destructive`, `muted`, `accent`, `primary`) — do not hardcode hex values
- Status colors:
  - Success/confirmed: `text-green-600` / `bg-green-50`
  - Warning/consecutive: `text-amber-600` / `bg-amber-50`
  - Error/destructive: use shadcn/ui `destructive` token
  - Draft/neutral: `text-muted-foreground` / `bg-muted`

---

## 5. Component Patterns

### Lists (members, schedules, loans)
- Use card-based lists on mobile, not tables
- Tables are acceptable on tablet/desktop (`md:` breakpoint and above) for data-heavy views
- Every list item must have a clear primary action on tap (navigate to detail)
- Secondary actions (edit, delete) go in a context menu or swipe action — never crowd the list item
- Empty states are mandatory: icon + short message + CTA if applicable
  ```
  Example: "Nenhum membro encontrado. Adicione o primeiro membro da sua igreja."
  [+ Adicionar Membro]
  ```

### Forms
- One column on mobile — never multi-column on screens below `md:`
- Labels above inputs — never placeholder-only labels (accessibility)
- Inline validation: show errors on blur, not on every keystroke
- Required fields: mark with `*` and explain at the top ("Campos com * são obrigatórios")
- Submit button: always at the bottom, full-width on mobile (`w-full`)
- Disabled state while submitting: show spinner inside button, disable the button
- Never reset the form on a failed submission — preserve what the user typed

### Modals and drawers
- Use **bottom sheets (drawers)** on mobile, not center modals — they are easier to reach with thumbs
- Use center modals only for confirmation dialogs (destructive actions)
- Confirmation dialogs must: name the thing being deleted, explain the consequence, and use a red destructive button
  ```
  "Remover João da escala?"
  "João será removido do culto de 04/05/2025. Esta ação não pode ser desfeita."
  [Cancelar] [Remover]  ← destructive red
  ```
- Maximum one primary action per modal

### Navigation
- Bottom navigation bar for the 4–5 main sections (mobile)
- Sidebar navigation for tablet/desktop
- Active state must be visually unambiguous
- Back navigation: always available, never trap the user in a screen

### Loading states
- Use skeleton screens for initial data loads — not spinners over blank pages
- Use spinner inside the triggering button for mutations (save, publish, confirm)
- Never show a full-page loading overlay unless strictly necessary

### Error states
- Network errors: inline banner at the top of the affected section, not a full-page error
- 404 (resource not found): dedicated empty state with explanation
- 500 (server error): friendly message + retry button, log the error to console
- Never show raw error codes or stack traces to the user

---

## 6. Feedback & Micro-interactions

Every user action must have immediate feedback:

| Action | Feedback |
|---|---|
| Tap a button | Visual press state (Tailwind `active:` variants) |
| Submit a form | Button becomes disabled + spinner |
| Successful mutation | Toast notification (bottom of screen, auto-dismiss 3s) |
| Failed mutation | Inline error message near the cause |
| Destructive action | Confirmation dialog before executing |
| Long operation (>1s) | Progress indicator |

Toast messages:
- Success: short, specific ("Escala publicada com sucesso.")
- Error: short + actionable ("Não foi possível salvar. Tente novamente.")
- Never say "Erro 500" or technical codes in toasts

---

## 7. Mobile-First Implementation Rules

Always build mobile layout first, then add `md:` and `lg:` overrides.

```tsx
// Correct: mobile-first
<div className="flex flex-col gap-4 md:flex-row md:gap-6">

// Wrong: desktop-first
<div className="flex flex-row gap-6 sm:flex-col">
```

Test every screen mentally at **375px width** (iPhone SE) before considering it done.

Scrolling:
- Vertical scroll is fine and expected
- Horizontal scroll is a bug unless it is a deliberate carousel/tab component
- Never let content overflow the viewport horizontally

---

## 8. Accessibility (a11y)

Non-negotiable minimums:
- All interactive elements are keyboard-navigable
- All images have `alt` text (or `alt=""` for decorative images)
- All form inputs have associated `<label>` elements
- Color is never the only differentiator
- Focus ring is always visible (do not remove `outline` without replacing it)
- Icon-only buttons have `aria-label`

shadcn/ui components are accessible by default — do not override their ARIA attributes without a clear reason.

---

## 9. Copy & Language

The app is in **Brazilian Portuguese**. Rules:
- Use "você" (not "tu")
- Use verbs in imperative for CTAs: "Adicionar", "Salvar", "Publicar", "Cancelar"
- Avoid jargon — write for a non-technical church volunteer
- Date format: `dd/MM/yyyy` (e.g., "04/05/2025")
- Time format: `HH:mm` (e.g., "14:30")
- Currency: not applicable in MVP
- Never expose technical terms (UUID, endpoint, cache, JWT) in UI copy

Section names (use these exact labels in navigation and headers):
| Domain | Label in UI |
|---|---|
| Agenda (pastoral) | Agenda |
| Schedules (worship) | Escala |
| Members | Membros |
| Inventory | Patrimônio |
| Notifications | Notificações |
| Profile | Meu Perfil |

---

## 10. What Good Looks Like

Before marking any screen as done, verify:
- [ ] Works at 375px width without horizontal scroll
- [ ] Every touch target is at least 48px
- [ ] Empty state is implemented
- [ ] Loading state is implemented (skeleton or button spinner)
- [ ] Error state is implemented
- [ ] Destructive actions have confirmation
- [ ] All text is in pt-BR
- [ ] No hardcoded colors — using Tailwind or shadcn/ui tokens
- [ ] No inline styles
- [ ] Form preserves data on failed submission