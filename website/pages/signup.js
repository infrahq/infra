import Head from 'next/head'
import { useState, useEffect, useRef } from 'react'
import { Transition } from '@headlessui/react'

import { useSignup } from '../lib/hooks'

export default function Index() {
  const [email, setEmail] = useState('')
  const [show, setShow] = useState(false)
  const inputRef = useRef()

  useEffect(() => setShow(true), [])

  const { submitted, success, error, submit, renderRecaptcha } = useSignup({
    email,
  })

  return (
    <>
      <Transition show={show}>
        <Head>
          <title>Infra - Signup</title>
          <meta
            property='og:title'
            content='Infrastructure Access'
            key='title'
          />
          <meta property='og:description' content='Signup for early access' />
        </Head>
        <div className='fixed inset-0 bg-black'>
          <div className='flex min-h-full items-end p-6 md:items-center'>
            <div className='flex w-full max-w-4xl flex-col px-10 py-10 text-white'>
              <Transition.Child
                enter='ease-out duration-300 delay-300'
                enterFrom='opacity-0 translate-y-10'
                enterTo='opacity-100 translate-y-0'
                leave='ease-in duration-200'
                leaveFrom='opacity-100 scale-100'
                leaveTo='opacity-0 scale-95'
              >
                <h1 className='mb-6 font-display text-4xl md:text-6xl'>
                  Sign up for early access
                </h1>
              </Transition.Child>
              <Transition.Child
                enter='ease-out duration-300 delay-[1000ms]'
                enterFrom='opacity-0 translate-y-10'
                enterTo='opacity-100 translate-y-0'
                leave='ease-in duration-200'
                leaveFrom='opacity-100 scale-100'
                leaveTo='opacity-0 scale-95'
              >
                <h2 className='mb-8 text-lg text-zinc-400 md:text-2xl'>
                  Infra is currently available in early access. <br />
                  Sign up and we&apos;ll reach out to you soon.
                </h2>
              </Transition.Child>
              <Transition.Child
                enter='ease-out duration-300 delay-[1500ms]'
                enterFrom='opacity-0 translate-y-10'
                enterTo='opacity-100 translate-y-0'
                leave='ease-in duration-200'
                leaveFrom='opacity-100 scale-100'
                leaveTo='opacity-0 scale-95'
                afterEnter={() => inputRef.current?.focus()}
              >
                {success ? (
                  <div className='py-2'>
                    Thanks for signing up. We&apos;ll be in touch!
                  </div>
                ) : (
                  <form
                    className='flex flex-col md:flex-row'
                    onSubmit={e => {
                      e.preventDefault()
                      submit()
                      return false
                    }}
                  >
                    <input
                      ref={inputRef}
                      type='email'
                      required
                      placeholder='your email'
                      className='mb-3 block w-full max-w-md rounded-lg border border-white/10 bg-white/5 py-4 px-5 text-lg font-medium text-white shadow-md shadow-black/5 placeholder:text-zinc-500 focus:border-white/25 focus:outline-none focus:ring-0 md:mb-0'
                      onChange={e => setEmail(e.target.value)}
                    />
                    <button
                      type='submit'
                      disabled={submitted}
                      className='mb-3 flex w-[160px] items-center justify-center whitespace-nowrap rounded-full bg-white px-10 py-2 text-lg font-medium text-black focus:outline-none disabled:pointer-events-none disabled:bg-zinc-100 md:ml-6 md:mb-0'
                    >
                      {submitted ? (
                        <svg
                          className='h-5 w-5 animate-spin text-black'
                          xmlns='http://www.w3.org/2000/svg'
                          fill='none'
                          viewBox='0 0 24 24'
                        >
                          <circle
                            className='opacity-25'
                            cx='12'
                            cy='12'
                            r='10'
                            strokeWidth='4'
                          ></circle>
                          <path
                            className='opacity-75'
                            fill='currentColor'
                            d='M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
                          ></path>
                        </svg>
                      ) : (
                        'Sign up'
                      )}
                    </button>
                    {error && (
                      <p className='absolute top-full text-sm text-red-400'>
                        An error occured. Please try again later.
                      </p>
                    )}
                    {renderRecaptcha()}
                  </form>
                )}
              </Transition.Child>
            </div>
          </div>
        </div>
      </Transition>
    </>
  )
}
