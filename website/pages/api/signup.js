import Analytics from 'analytics-node'

const SENDGRID_API_KEY = process.env.SENDGRID_API_KEY
const RECAPTCHA_SECRET_KEY = process.env.RECAPTCHA_SECRET_KEY
const SENDGRID_LIST_ID = process.env.SENDGRID_LIST_ID

const analytics = new Analytics(process.env.NEXT_PUBLIC_SEGMENT_WRITE_KEY)

export default async function signup(req, res) {
  if (req.method !== 'POST') {
    res.status(405).json({ error: 'method must be POST' })
    return
  }

  if (!req.body.email || !req.body.code) {
    res.status(400).json({ error: 'invalid email or code' })
    return
  }

  if (!SENDGRID_API_KEY || !SENDGRID_LIST_ID) {
    console.error('server not configured')
    res.status(500).end()
    return
  }

  try {
    const response = await fetch(
      'https://www.google.com/recaptcha/api/siteverify ',
      {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          secret: RECAPTCHA_SECRET_KEY,
          response: req.body.code,
          remoteip: req.ip,
        }),
      }
    )

    const data = await response.json()

    if (!data || !data.success) {
      console.error('could not verify recaptcha')
      res.status(403).end()
      return
    }
  } catch (e) {
    console.error('error verifying recaptcha')
    res.status(500).end()
    return
  }

  if (req.body.aid && process.env.NEXT_PUBLIC_SEGMENT_WRITE_KEY) {
    analytics.track({
      anonymousId: req.body.aid,
      event: 'website:signup',
      traits: {
        email: req.body.email,
      },
    })
  }

  try {
    await fetch('https://api.sendgrid.com/v3/marketing/contacts', {
      method: 'PUT',
      headers: {
        Authorization: `BEARER ${SENDGRID_API_KEY}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        contacts: [
          {
            email: req.body.email,
          },
        ],
        list_ids: [SENDGRID_LIST_ID],
      }),
    })
  } catch (e) {
    console.error('could not subscribe')
    res.status(500).end()
    return
  }

  res.status(200).end()
}
