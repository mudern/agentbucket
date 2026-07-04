import { createContext, useContext, useState, useCallback } from 'react'

const supportedLangs = ['zh', 'en', 'fr', 'ja', 'de', 'ko', 'es', 'ar', 'pt', 'it']
const defaultLang = 'zh'

const langNames = {
  zh: '中文', en: 'EN', fr: 'FR', ja: '日本語',
  de: 'DE', ko: '한국어', es: 'ES', ar: 'العربية',
  pt: 'PT', it: 'IT',
}

const LanguageContext = createContext()

export function LanguageProvider({ children }) {
  const [lang, setLangState] = useState(() => {
    const stored = localStorage.getItem('agentbucket.lang')
    return stored && supportedLangs.includes(stored) ? stored : defaultLang
  })

  const setLang = useCallback((newLang) => {
    if (supportedLangs.includes(newLang)) {
      localStorage.setItem('agentbucket.lang', newLang)
      setLangState(newLang)
    }
  }, [])

  return (
    <LanguageContext.Provider value={{ lang, setLang }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const ctx = useContext(LanguageContext)
  if (!ctx) throw new Error('useLanguage must be used within LanguageProvider')
  return ctx
}

let _translations = null

export async function loadTranslations() {
  if (_translations) return _translations
  _translations = {
    zh: (await import('./zh.js')).default,
    en: (await import('./en.js')).default,
    fr: (await import('./fr.js')).default,
    ja: (await import('./ja.js')).default,
    de: (await import('./de.js')).default,
    ko: (await import('./ko.js')).default,
    es: (await import('./es.js')).default,
    ar: (await import('./ar.js')).default,
    pt: (await import('./pt.js')).default,
    it: (await import('./it.js')).default,
  }
  return _translations
}

export function useT() {
  const { lang } = useLanguage()
  return useCallback((key, fallback) => {
    if (!_translations || !_translations[lang]) return fallback || key
    const keys = key.split('.')
    let val = _translations[lang]
    for (const k of keys) {
      if (val == null) break
      val = val[k]
    }
    return val ?? fallback ?? key
  }, [lang])
}

export { langNames, supportedLangs }
