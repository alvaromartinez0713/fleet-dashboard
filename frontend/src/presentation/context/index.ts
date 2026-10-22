/**
 * Context module - Provides application-level context and hooks.
 *
 * Exports:
 * - ApplicationContext: React Context for application-level services
 * - ApplicationContextType: Type definition for context value
 * - ContextProvider: The original ApplicationProvider from ApplicationContext
 * - ApplicationProvider: Wrapper ApplicationProvider that creates dependencies
 * - useApplicationContext: Hook to access context
 */

export {
  ApplicationContext,
  ApplicationContextType,
  ApplicationProvider as ContextProvider,
  ApplicationProviderProps,
  useApplicationContext,
} from './ApplicationContext';

export { ApplicationProvider } from './ApplicationProvider';
