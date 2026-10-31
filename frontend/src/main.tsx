import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/inter'
import './index.css'
import App from './App.tsx'
import { ToastProvider } from './components/ToastProvider.tsx'

// Apply the saved theme before the first paint, including on the sign-in pages.
document.documentElement.classList.toggle('dark', localStorage.getItem('theme') === 'dark')

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ToastProvider>
      <App />
    </ToastProvider>
  </StrictMode>,
)
