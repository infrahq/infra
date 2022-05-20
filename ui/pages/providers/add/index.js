import Link from 'next/link'
import Head from 'next/head'

import { providers } from '../../../lib/providers'

import Fullscreen from '../../../components/layouts/fullscreen'

function Provider ({ kind, name, available }) {
  return (
    <div className={`rounded-lg px-3 py-4 flex items-center select-none bg-transparent border border-gray-800 ${available ? 'hover:border-gray-600 cursor-pointer' : 'opacity-40 grayscale select-none'}`}>
      <img className='flex-none w-6 mr-4' src={`/providers/${kind}.svg`} />
      <div>
        <h3 className='flex-1 text-xs'>{name}</h3>
        <h4 className='text-[10px] text-gray-400'>{available ? 'Identity Provider' : 'Coming Soon'}</h4>
      </div>
    </div>
  )
}

export default function ProvidersAdd () {
  return (
    <div className='pt-8 px-1 pb-1'>
      <Head>
        <title>Add Identity Provider</title>
      </Head>
      <header className='flex flex-row px-4 text-xs'>
        <img src='/providers.svg' className='w-6 h-6 mr-2 mt-0.5' />
        <div>
          <h1>Connect an Identity Provider</h1>
          <h2 className='text-gray-400'>Select an identity provider to continue</h2>
        </div>
      </header>
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
  )
}

ProvidersAdd.layout = page => <Fullscreen closeHref='/providers'>{page}</Fullscreen>
