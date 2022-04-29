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
        <h4 className='text-sm text-secondary'>{available ? 'Identity Provider' : 'Coming Soon'}</h4>
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
      <div className='flex flex-col mb-24'>
        <div className='flex my-4 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full mx-auto'>
          <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
            <img className='w-8 h-8' src='/providers-color.svg' />
          </div>
        </div>
        <h1 className='text-base font-bold mb-1 text-center'>Add Identity Provider</h1>
        <h2 className='text-secondary mb-4 text-sm max-w-xs mx-auto text-center'>Select an identity provider to continue</h2>
        <div className='grid grid-cols-2 lg:grid-cols-3 gap-1 my-8'>
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
    </FullscreenModal>
  )
}
