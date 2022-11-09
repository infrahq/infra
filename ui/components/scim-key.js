import { useState } from 'react'
import Link from 'next/link'

import copy from 'copy-to-clipboard'
import { DocumentDuplicateIcon, CheckIcon } from '@heroicons/react/24/outline'

export default function SCIMKey({ accessKey, errorMsg }) {
  const [keyCopied, setKeyCopied] = useState(false)

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>SCIM Access Key</h1>
      <div className='space-y-4'>
        <>
          {errorMsg === '' ? (
            <>
              <div className='mb-2'>
                <p className='mt-1 text-sm text-gray-500'>
                  Use this access key to configure your identity provider for
                  inbound SCIM provisioning
                </p>
              </div>
              <div className='group relative my-4 flex'>
                <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
                  {accessKey}
                </pre>
                <button
                  className='absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70'
                  type='button'
                  onClick={() => {
                    copy(accessKey)
                    setKeyCopied(true)
                    setTimeout(() => setKeyCopied(false), 2000)
                  }}
                >
                  {keyCopied ? (
                    <CheckIcon className='h-4 w-4 text-green-500' />
                  ) : (
                    <DocumentDuplicateIcon className='h-4 w-4' />
                  )}
                </button>
              </div>
            </>
          ) : (
            <div
              class='mb-4 rounded-lg bg-red-100 p-4 text-sm text-red-700'
              role='alert'
            >
              <span class='font-medium'>Error:</span> {errorMsg}
            </div>
          )}
        </>

        {/* Finish */}
        <div className='my-10 flex justify-end'>
          <Link href='/settings?tab=providers'>
            <a className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
              Finish
            </a>
          </Link>
        </div>
      </div>
    </div>
  )
}
