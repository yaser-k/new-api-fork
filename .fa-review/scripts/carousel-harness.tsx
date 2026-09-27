// Carousel harness: the carousel is not used by any page, so it is shown here
// with the real component and the app's built stylesheet.
import { createRoot } from 'react-dom/client'

import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from '@/components/ui/carousel'
import { DirectionProvider } from '@/components/ui/direction'

function Demo(props: { dir: 'ltr' | 'rtl'; label: string }) {
  return (
    <DirectionProvider direction={props.dir}>
      <section dir={props.dir} lang={props.dir === 'rtl' ? 'fa' : 'en'} style={{ margin: '32px 96px' }}>
        <h2 className='mb-3 text-sm font-semibold'>{props.label}</h2>
        <Carousel style={{ width: 520 }} opts={{ startIndex: 1 }}>
          <CarouselContent>
            {['۱', '۲', '۳'].map((n, i) => (
              <CarouselItem key={n} style={{ flexBasis: '50%' }}>
                <div className='bg-muted flex items-center justify-center rounded-lg border text-2xl' style={{ height: 112 }}>
                  {props.dir === 'rtl' ? `اسلاید ${n}` : `Slide ${i + 1}`}
                </div>
              </CarouselItem>
            ))}
          </CarouselContent>
          <CarouselPrevious />
          <CarouselNext />
        </Carousel>
      </section>
    </DirectionProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <>
    <Demo dir='rtl' label='dir=rtl (Persian)' />
    <Demo dir='ltr' label='dir=ltr (English, unchanged)' />
  </>
)
