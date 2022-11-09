import { useRouter } from 'next/router'
import Tippy from '@tippyjs/react'

import { providers as providersList } from '../lib/providers'

function oidcLogin(
  { id, clientID, authURL, scopes, kind, callbackPath },
  next
) {
  window.localStorage.setItem('providerID', id)
  window.localStorage.setItem('providerKind', kind)
  if (next) {
    window.localStorage.setItem('next', next)
  }

  const state = [...Array(10)]
    .map(() => (~~(Math.random() * 36)).toString(36))
    .join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + callbackPath
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `${authURL}?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=${scopes.join(
    '+'
  )}&state=${state}`
}

export default function Providers({ providers, buttonPrompt, callbackPath }) {
  const router = useRouter()
  const { next } = router.query
  return (
    <>
      <div className='mt-4 w-full text-sm'>
        {providers.map(
          p =>
            p.kind && (
              <Tippy
                content={`${p.name} — ${p.url}`}
                className='whitespace-no-wrap z-8 relative w-auto rounded-md bg-black p-2 text-xs text-white shadow-lg'
                interactive={true}
                interactiveBorder={20}
                offset={[0, 5]}
                delay={[250, 0]}
                placement='top'
              >
                <button
                  onClick={() => oidcLogin({ ...p, callbackPath }, next)}
                  key={p.id}
                  className='my-2 inline-flex w-full items-center rounded-md border border-gray-300 bg-white py-2.5 px-4 text-gray-500 shadow-sm hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
                >
                  <img
                    alt='identity provider icon'
                    className='h-4'
                    src={`/providers/${p.kind}.svg`}
                  />
                  <span className='items-center truncate pl-4 text-gray-800'>
                    {providersList.filter(i => i.kind === p.kind) ? (
                      <div className='truncate'>
                        <span>
                          {buttonPrompt} {p.name}
                        </span>
                      </div>
                    ) : (
                      'Single Sign-On'
                    )}
                  </span>
                </button>
              </Tippy>
            )
        )}
      </div>
    </>
  )
}
