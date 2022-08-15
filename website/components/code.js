/* global Prism */
import React, { useState, useEffect, useRef } from 'react'
import copy from 'copy-to-clipboard'
import { Square2StackIcon, CheckIcon } from '@heroicons/react/24/outline'
import 'prismjs'
import 'prismjs/components/prism-yaml.min'
import 'prismjs/components/prism-bash.min'

export default function Code({ children, language = 'none' }) {
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
  }, [copied, children])

  return (
    <div className='group relative my-4 flex'>
      <pre ref={ref} className={`language-${language}`}>
        {children}
      </pre>
      <button
        onClick={() => setCopied(true)}
        className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white/40 px-2 py-2 text-black/40 opacity-0 backdrop-blur-xl hover:text-black/70 ${
          copied ? 'opacity-100' : 'group-hover:opacity-100'
        }`}
      >
        {copied ? (
          <CheckIcon className='h-3 w-4 text-green-500' />
        ) : (
          <Square2StackIcon className='h-4 w-4' />
        )}
      </button>
    </div>
  )
}
