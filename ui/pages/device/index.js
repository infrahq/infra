import { useRouter } from 'next/router'
import React, { useState } from 'react'

import Dashboard from '../../components/layouts/dashboard'

export default function Device() {
  const router = useRouter()
  const [code, setCode] = useState(router.query.code)
  const [codeEntered, setCodeEntered] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e) {
    e.preventDefault()

    if (code.length == 9) setCode(code.substring(0, 4) + code.substring(5, 9))

    try {
      const res = await fetch('/api/device/approve', {
        method: 'post',
        body: JSON.stringify({
          userCode: code,
        }),
      })

      await jsonBody(res)

      setCodeEntered(true)
    } catch (e) {
      setError(e.message)
    }

    return false
  }

  async function setCodeSegment(segment, pos) {
    var codeCopy = code || ''
    for (; codeCopy.length < 8; ) {
      codeCopy += ' '
    }
    codeCopy =
      codeCopy.substring(0, pos) +
      segment +
      codeCopy.substring(pos + 1, codeCopy.length)
    codeCopy = codeCopy.trimEnd()
    codeCopy = codeCopy.substring(0, 8)
    setCode(codeCopy.toUpperCase())
  }

  async function processKeyDown(e, pos) {
    const lastField = document.querySelector(`input[name=code${pos - 1}]`)
    const nextField = document.querySelector(`input[name=code${pos + 1}]`)
    switch (e.key) {
      case 'Escape':
        break
      case 'Backspace':
        e.preventDefault()
        setCodeSegment('', pos)
        break
      case 'ArrowLeft':
        if (lastField !== null) {
          lastField.focus()
          lastField.selectionStart = 0
          lastField.selectionEnd = 1
        }
        break
      case 'ArrowRight':
        if (nextField !== null) {
          nextField.focus()
          nextField.selectionStart = 0
          nextField.selectionEnd = 1
        }
        break
      case 'Meta':
      case 'Control':
      case 'Alt':
        e.preventDefault()
        break
      case 'Tab':
      case 'Shift':
        break
      case 'Enter':
        return
      default:
        e.preventDefault()
        if (e.code >= 'KeyA' && e.code <= 'KeyZ') {
          setCodeSegment(e.key, pos)
          if (nextField !== null) {
            nextField.focus()
            nextField.selectionStart = 0
            nextField.selectionEnd = 1
          }
        }
    }
  }

  return (
    <div className='flex min-h-[280px] w-full flex-col items-center px-10 py-8'>
      <>
        <h1 className='text-base font-bold leading-snug'>Confirm Log In</h1>
        {codeEntered ? (
          <p className='text-s my-3 flex max-w-[260px] flex-1 flex-col items-center justify-center text-center text-gray-600'>
            You are logged in. You may now close this window.
          </p>
        ) : (
          <>
            <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
              Please confirm this is the code displayed in your terminal.
            </h2>
            <form
              onSubmit={onSubmit}
              className='relative flex w-full max-w-sm flex-1 flex-col justify-center'
            >
              <div className='my-2 flex w-full place-content-center'>
                {'01234567'.split('').map((k, i) => (
                  <React.Fragment key={i}>
                    <input
                      required
                      autoFocus={
                        i ==
                        Math.max((code || '').replace('-', '').length - 1, 0)
                      }
                      type='text'
                      name={'code' + i}
                      defaultValue={(code || '')
                        .replace('-', '')
                        .substring(i, i + 1)}
                      onKeyDown={e => processKeyDown(e, i)}
                      className={`focus:border-blue-80 focus:ring-blue-80 xs:w-8  mr-1 w-8 rounded-md pl-0 pr-0 text-center uppercase shadow-sm sm:w-10 lg:text-lg ${
                        error ? 'border-red-500' : 'border-gray-300'
                      }`}
                    />
                    {i == 3 ? <div className='mt-3 ml-3 mr-4'>-</div> : ''}
                  </React.Fragment>
                ))}
                {error && (
                  <p className='absolute top-full mt-1 text-xs text-red-500'>
                    Error: {error}. Code may be expired.
                  </p>
                )}
              </div>
              <button
                type='submit'
                disabled={codeEntered}
                className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed  disabled:opacity-30 sm:text-sm'
              >
                Confirm and Authorize New Device
              </button>
            </form>
          </>
        )}
      </>
    </div>
  )
}

Device.layout = page => <Dashboard>{page}</Dashboard>
