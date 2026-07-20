class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class PageIterator:
    def __init__(self,pages):self.pages=pages;self.page_index=0;self.items=[];self.item_index=0;self.fetched=0
    def __next__(self):
        while self.item_index>=len(self.items):
            if self.page_index>=len(self.pages):raise StopIteration
            page=self.pages[self.page_index];self.page_index+=1
            if isinstance(page,dict):raise PatternError(page["error"])
            self.items=list(page);self.item_index=0;self.fetched+=1
        value=self.items[self.item_index];self.item_index+=1;return value
def evaluate(input_data):
    iterator=PageIterator(input_data.get("pages",[]));items=[];exhausted=False
    for _ in range(input_data.get("take",0)):
        try:items.append(next(iterator))
        except StopIteration:exhausted=True;break
    return{"items":items,"fetched_pages":iterator.fetched,"exhausted":exhausted}
