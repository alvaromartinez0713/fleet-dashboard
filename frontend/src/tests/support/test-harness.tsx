/**
 * Test Harness for VehicleDashboard Integration Tests
 *
 * Isolates framework coupling in one place per testing.md section 6.
 * All tests use this harness, not ApplicationProvider directly.
 *
 * This is the ONLY place that knows about:
 * - ApplicationProvider
 * - MSW setup
 * - React Testing Library configuration
 *
 * If we switch providers or frameworks, only this file changes.
 * Tests remain unchanged.
 */

import React from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { ApplicationProvider } from '@presentation/context/ApplicationProvider';

/**
 * Custom render function that wraps components with ApplicationProvider
 *
 * Usage in tests:
 * ```typescript
 * render(<VehicleDashboard />, { ...options });
 * // Equivalent to:
 * // render(
 * //   <ApplicationProvider>
 * //     <VehicleDashboard />
 * //   </ApplicationProvider>
 * // );
 * ```
 *
 * This abstracts away the implementation detail that we need ApplicationProvider.
 * Tests only call render(); they don't know about the provider.
 */
export const renderWithApplicationContext = (
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>,
) => {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <ApplicationProvider>{children}</ApplicationProvider>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};

/**
 * Alias for convenience
 */
export const renderInApp = renderWithApplicationContext;
