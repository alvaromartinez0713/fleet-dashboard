/**
 * MSW Service Worker Configuration for Browser
 * 
 * This file is imported in main.tsx to initialize MSW in the browser.
 * It uses a Service Worker to intercept HTTP requests at the network level.
 */

import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

export const worker = setupWorker(...handlers);
