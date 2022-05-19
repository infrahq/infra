import Link from 'next/link'
import Head from 'next/head'

import { providers } from '../../../lib/providers'

import FullscreenModal from '../../../components/modals/fullscreen'

function Provider ({ kind, name, available }) {
  return (
    <div className={`rounded-lg px-3 py-4 flex items-center select-none bg-transparent border border-gray-800 ${available ? 'hover:border-gray-500 cursor-pointer' : 'opacity-40 grayscale select-none'}`}>
      <img className='flex-none w-6 mr-4' src={`/providers/${kind}.svg`} />
      <div>
        <h3 className='flex-1 text-xs'>{name}</h3>
        <h4 className='text-[10px] text-gray-400'>{available ? 'Identity Provider' : 'Coming Soon'}</h4>
      </div>
    </div>
  )
}

export default function () {
  return (
    <FullscreenModal closeHref='/providers'>
      <Head>
        <title>Add Identity Provider</title>
      </Head>
      <div className='w-full max-w-xs'>
        <div className='flex flex-col pt-8 px-1 pb-1 border rounded-lg border-gray-800'>
          <div className='flex flex-row space-x-2 items-center px-3'>
            <img src='/providers.svg' className='w-6 h-6 mr-1' />
            <div>
              <h1 className='text-xs mb-0.5'>Connect an Identity Provider</h1>
              <h2 className='text-xs text-gray-400'>Select an identity provider to continue</h2>
            </div>
          </div>
          <div className='flex flex-col mt-11 space-y-1'>
            {providers.map(p => (
              p.available
                ? (
                  <Link key={p.name} href={`/providers/add/details?kind=${p.kind}`}>
                    <a>
                      <Provider {...p} />
                    </a>
                  </Link>
                  )
                : (
                  <Provider key={p.name} {...p} />
                  )
            ))}
          </div>
        </div>
      </div>
    </FullscreenModal>
  )
}
