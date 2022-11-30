import { useRouter } from 'next/router'
import Tippy from '@tippyjs/react'
import Cookies from 'universal-cookie'

import { currentBaseDomain } from '../lib/login'
import { providers as providersList } from '../lib/providers'

export function oidcLogin(
  { baseDomain, loginDomain, id, clientID, authURL, scopes },
  next
) {
  if (baseDomain === '') {
    // this is possible if not configured on the server
    // fallback to the browser domain
    baseDomain = currentBaseDomain()
  }

  let redirectURL = window.location.origin + '/login/callback'
  if (id === '') {
    // managed oidc providers (social login) need to be sent to the base redirect URL before they are redirected to org login
    const cookies = new Cookies()
    cookies.set('finishLogin', window.location.host, {
      path: '/',
      domain: `.${baseDomain}`,
      sameSite: 'lax',
    })
    redirectURL = window.location.protocol + '//' + loginDomain + '/redirect' // go to the social login redirect specified by the server
  }

  oidc(id, clientID, authURL, scopes, redirectURL, next)
}

export function oidcSignup({ id, clientID, authURL, scopes }, next) {
  const redirectURL = window.location.origin + '/signup/callback'
  oidc(id, clientID, authURL, scopes, redirectURL, next)
}

function oidc(id, clientID, authURL, scopes, redirectURL, next) {
  window.localStorage.setItem('redirectURL', redirectURL)

  window.localStorage.setItem('providerID', id)
  if (next) {
    window.localStorage.setItem('next', next)
  }

  const state = [...Array(10)]
    .map(() => (~~(Math.random() * 36)).toString(36))
    .join('')
  window.localStorage.setItem('state', state)

  document.location.href = `${authURL}?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=${scopes.join(
    '+'
  )}&state=${state}`
}

export default function Providers({
  buttonPrompt,
  authnFunc,
  baseDomain,
  loginDomain,
  providers,
}) {
  const router = useRouter()
  const { next } = router.query
  return (
    <>
      <div className='mt-4 w-full text-sm'>
        {providers.map(
          p =>
            p.kind && (
              <div key={p.id}>
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
                    onClick={() =>
                      authnFunc({ baseDomain, loginDomain, ...p }, next)
                    }
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
              </div>
            )
        )}
      </div>
    </>
  )
}
