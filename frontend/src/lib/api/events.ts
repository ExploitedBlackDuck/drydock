// Event names mirror the typed constants emitted by the Go side (app/events.go).
// Keeping them in one module is the frontend half of the "no magic strings
// across boundaries" rule (PROJECT-BOOK §2.7).
export const AppReady = 'app:ready';
