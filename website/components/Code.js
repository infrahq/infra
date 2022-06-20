/* global Prism */
import React, { useState, useEffect, useRef } from 'react'
import copy from 'copy-to-clipboard'
import { DuplicateIcon, CheckIcon } from '@heroicons/react/outline'
import 'prismjs'
import 'prismjs/components/prism-yaml.min'
import 'prismjs/components/prism-bash.min'

export default function ({ children, language = 'none' }) {
  const [copied, setCopied] = useState(false)
  const ref = useRef(null)

  useEffect(() => {
    if (ref.current) {
      Prism.highlightElement(ref.current, false)
    }
  }, [children])

  useEffect(() => {
    if (copied) {
      copy(children)
      const to = setTimeout(setCopied, 1000, false)
      return () => clearTimeout(to)
    }
  }, [copied])

  return (
    <div className='relative group flex my-4'>
      <pre ref={ref} className={`language-${language}`}>
        {children}
      </pre>
      <button onClick={() => setCopied(true)} className='opacity-0 group-hover:opacity-100 absolute right-2.5 top-2.5 px-2 py-2 rounded-md bg-white/5 backdrop-blur-xl border border-white/10 text-white/50 hover:text-white/70'>
        {copied ? <CheckIcon className='w-4 h-4 text-green-300' /> : <DuplicateIcon className='w-4 h-4' />}
      </button>
    </div>
  )
}
