import Link from 'next/link'
import Head from 'next/head'

import { providers } from '../../../lib/providers'

import Fullscreen from '../../../components/layouts/fullscreen'

function Provider({ kind, name, available }) {
  return (
    <div
      className={`flex select-none items-center rounded-lg border border-gray-800 bg-transparent px-3 py-4 ${
        available
          ? 'cursor-pointer hover:border-gray-600'
          : 'select-none opacity-40 grayscale'
      }`}
    >
      <img
        alt='provider icon'
        className='mr-4 w-6 flex-none'
        src={`/providers/${kind}.svg`}
      />
      <div>
        <h3 className='flex-1 text-2xs'>{name}</h3>
        <h4 className='text-[10px] text-gray-400'>
          {available ? 'Identity Provider' : 'Coming Soon'}
        </h4>
      </div>
    </div>
  )
}

export default function ProvidersAdd() {
  return (
    <div className='px-1 pt-8 pb-1'>
      <Head>
        <title>Add Identity Provider</title>
      </Head>
      <header className='flex flex-row px-4 text-2xs'>
        <img
          alt='providers icon'
          src='/providers.svg'
          className='mr-2 mt-0.5 h-6 w-6'
        />
        <div>
          <h1>Connect an Identity Provider</h1>
          <h2 className='text-gray-400'>
            Select an identity provider to continue
          </h2>
        </div>
      </header>
      <div className='mt-11 flex flex-col space-y-1'>
        {providers.map(p =>
          p.available ? (
            <Link key={p.name} href={`/providers/add/details?kind=${p.kind}`}>
              <a>
                <Provider {...p} />
              </a>
            </Link>
          ) : (
            <Provider key={p.name} {...p} />
          )
        )}
      </div>
    </div>
  )
}

ProvidersAdd.layout = page => (
  <Fullscreen closeHref='/providers'>{page}</Fullscreen>
)
