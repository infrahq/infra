import React, { useState, useEffect } from 'react'
import copy from 'copy-to-clipboard'
import { DuplicateIcon, CheckIcon } from '@heroicons/react/outline'
import Highlight, { defaultProps } from 'prism-react-renderer'
import 'prismjs'
import 'prismjs/components/prism-yaml.min'
import 'prismjs/components/prism-bash.min'

const theme = {
  plain: {
    color: '#F8F8F2'
  },
  styles: [{
    types: ['prolog', 'constant', 'builtin'],
    style: {
      color: 'rgb(189, 147, 249)'
    }
  }, {
    types: ['inserted', 'function'],
    style: {
      color: 'rgb(80, 250, 123)'
    }
  }, {
    types: ['deleted'],
    style: {
      color: 'rgb(255, 85, 85)'
    }
  }, {
    types: ['changed'],
    style: {
      color: 'rgb(255, 184, 108)'
    }
  }, {
    types: ['punctuation', 'symbol'],
    style: {
      color: 'hsl(220, 14%, 71%)'
    }
  }, {
    types: ['string', 'char', 'tag', 'selector'],
    style: {
      color: 'rgb(255, 121, 198)'
    }
  }, {
    types: ['keyword', 'variable'],
    style: {
      color: 'hsl(207, 82%, 66%)'
    }
  }, {
    types: ['comment'],
    style: {
      color: '#6D7583'
    }
  }, {
    types: ['attr-name'],
    style: {
      color: 'rgb(241, 250, 140)'
    }
  }, {
    types: ['important', 'atrule', 'rule'],
    style: {
      color: 'hsl(286, 60%, 67%)'
    }
  }]
}

export default function ({ children, language = 'none' }) {
  const [copied, setCopied] = useState(false)
  useEffect(() => {
    if (copied) {
      copy(children)
      const to = setTimeout(setCopied, 1000, false)
      return () => clearTimeout(to)
    }
  }, [copied])

  return (
    <div className='relative group'>
      <Highlight {...defaultProps} theme={theme} code={children.replace(/\n$/, '')} language={language}>
        {({ className, style, tokens, getLineProps, getTokenProps }) => (
          <pre className={className} style={style}>
            {tokens.map((line, i) => (
              <div key={i} {...getLineProps({ line, key: i })}>
                {line.map((token, key) => (
                  <span key={key} {...getTokenProps({ token, key })} />
                ))}
              </div>
            ))}
          </pre>
        )}
      </Highlight>
      <button onClick={() => setCopied(true)} className='opacity-0 group-hover:opacity-100 absolute right-2.5 top-2.5 px-2 py-2 rounded-md bg-white/5 backdrop-blur-xl border border-white/10 text-white/50 hover:text-white/70'>
        {copied ? <CheckIcon className='w-4 h-4 text-green-300' /> : <DuplicateIcon className='w-4 h-4' />}
      </button>
    </div>
  )
}
