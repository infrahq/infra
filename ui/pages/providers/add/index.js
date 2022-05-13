import Link from 'next/link'
import Head from 'next/head'

import { providers } from '../../../lib/providers'

import FullscreenModal from '../../../components/modals/fullscreen'

function Provider ({ kind, name, available }) {
  return (
    <div className={`rounded-xl px-6 py-4 flex items-center select-none bg-purple-100/5 ${available ? 'hover:bg-purple-100/10 cursor-pointer' : 'opacity-50 grayscale select-none'}`}>
      <img className='flex-none w-8 mr-4' src={`/providers/${kind}.svg`} />
      <div>
        <h3 className='flex-1'>{name}</h3>
        <h4 className='text-sm text-gray-300'>{available ? 'Identity Provider' : 'Coming Soon'}</h4>
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
      <div className='w-full max-w-sm'>
        <div className='flex flex-col py-8 px-4 border rounded-lg border-gray-950'>
          <div className='flex flex-row space-x-2 items-center'>
            <img src='/providers.svg' className='w-6 h-6' />
            <div>
              <h1 className='text-[12px] leading-[4px] tracking-tight'>Connect an Identity Provider</h1>
              <h2 className='text-[12px] leading-[4px] text-gray-400 mt-3'>Select an identity provider to continue</h2>
            </div>
          </div>
          <div className='flex flex-col mt-12'>
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
