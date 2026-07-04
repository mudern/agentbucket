import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { LanguageProvider, loadTranslations } from './i18n'
import { ToastProvider } from './components/Toast'
import './styles.css'
import 'highlight.js/styles/github-dark.css'

// Preload translations before rendering
loadTranslations().then(() => {
  ReactDOM.createRoot(document.getElementById('root')).render(
    <React.StrictMode>
      <BrowserRouter>
        <LanguageProvider>
          <ToastProvider>
            <App />
          </ToastProvider>
        </LanguageProvider>
      </BrowserRouter>
    </React.StrictMode>,
  )
})
